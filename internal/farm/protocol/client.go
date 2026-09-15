// Package protocol implements the QQ Farm WSS gateway client skeleton.
package protocol

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/gatepb"
)

// Encryptor encrypts/decrypts RPC bodies (typically TSDK ba/ca).
type Encryptor interface {
	Encrypt(buf []byte) ([]byte, error)
	Decrypt(buf []byte) ([]byte, error)
}

// HeartbeatTicker is optionally implemented by Encryptor (e.g. tsdk.Runtime) for ACE M().
type HeartbeatTicker interface {
	HeartbeatTick() error
}

// NotifyHandler receives server push frames (MessageTypeNotify).
type NotifyHandler func(service, method string, body []byte)

// GatewayBusyError marks a background request that yielded because the
// gateway had no capacity (bot GatewayBusyError).
type GatewayBusyError struct{ Message string }

func (e *GatewayBusyError) Error() string { return e.Message }

// IsGatewayYieldError mirrors bot isGatewayYieldError: whole-round yielding
// errors for background tasks.
func IsGatewayYieldError(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := err.(*GatewayBusyError); ok {
		return true
	}
	msg := err.Error()
	return contains(msg, "已让路") ||
		contains(msg, "stage=queued") ||
		contains(msg, "请求等待队列已满") ||
		contains(msg, "请求已中断") ||
		contains(msg, "连接未打开") ||
		contains(msg, "尚未登录")
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// queuedRequest is one entry in the dispatch queue (bot QueuedRequest).
type queuedRequest struct {
	service     string
	method      string
	body        []byte
	class       RequestClass
	lane        CriticalLane
	enqueuedAt  time.Time
	seq         int64 // 0 = still queued
	settled     bool
	ch          chan rpcResult
	expectReply bool
	timeout     time.Duration
	timer       *time.Timer
	queueWait   *time.Timer
}

// Client is a WSS gateway client.
type Client struct {
	url            string
	header         http.Header
	encryptor      Encryptor
	heartbeatEvery time.Duration
	onNotify       NotifyHandler
	onDisconnect   func(error)

	mu         sync.Mutex
	conn       *websocket.Conn
	clientSeq  int64
	serverSeq  int64
	pending    map[int64]*queuedRequest
	queue      []*queuedRequest
	closed     atomic.Bool
	hbCancel   context.CancelFunc
	readCancel context.CancelFunc

	// liveness / load snapshot (bot lastInboundAt + heartbeatMissCount)
	lastInboundAt     time.Time
	heartbeatMisses   int
	lastPressureLogAt time.Time

	// 出站 token 提供器：登录后暂存一次性 TSDK 初始化凭据，由下一条消息
	// 携带（对齐 rust Gateway.token_provider / bot GatewayTokenProvider）。
	tokens *GatewayTokenProvider
	// rebuilding 标记：TSDK 重建期间置位，心跳判死静默阈值放宽到 90s
	// （对齐 rust Gateway.rebuilding / begin_rebuild / end_rebuild）。
	rebuilding atomic.Bool
}

type rpcResult struct {
	body []byte
	meta *gatepb.Meta
	err  error
}

// Options configures Client.
type Options struct {
	URL            string
	Header         http.Header
	Encryptor      Encryptor // optional; if nil, bodies are sent plaintext
	HeartbeatEvery time.Duration
	OnNotify       NotifyHandler
	// OnDisconnect is invoked once when the read loop dies unexpectedly
	// (not after an intentional Close). Callers should stop the session;
	// WeChat accounts with Yingyongbao auth mint a new code instead of reusing the old one.
	OnDisconnect func(error)
}

// NewClient builds a disconnected client.
func NewClient(opts Options) *Client {
	if opts.HeartbeatEvery <= 0 {
		opts.HeartbeatEvery = 25 * time.Second
	}
	return &Client{
		url:            opts.URL,
		header:         opts.Header,
		encryptor:      opts.Encryptor,
		heartbeatEvery: opts.HeartbeatEvery,
		onNotify:       opts.OnNotify,
		onDisconnect:   opts.OnDisconnect,
		pending:        make(map[int64]*queuedRequest),
		clientSeq:      1,
		lastInboundAt:  time.Now(),
		tokens:         NewGatewayTokenProvider(),
	}
}

// Connect dials the gateway WebSocket and starts the read loop.
// Heartbeat must NOT start until after Login (matches qq-farm-bot).
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return nil
	}
	if c.header == nil {
		c.header = http.Header{}
	}
	if c.header.Get("Origin") == "" {
		c.header.Set("Origin", "https://gate-obt.nqf.qq.com")
	}
	if c.header.Get("User-Agent") == "" {
		c.header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows WindowsWechat/WMPF WindowsWechat(0x63090a13)")
	}
	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
		// 对齐 rust network/client.rs connect（tcp.set_nodelay(false)）：bot 跑在
		// Node 上，socket 默认开启 Nagle，小帧按内核节奏合批发送；Go net 默认
		// TCP_NODELAY=true（逐帧立发）。这里在拨号回调里对 TCP 连接显式关闭
		// nodelay，对齐 bot 的发送节奏，避免与 bot 不同的逐帧立发 TCP 分段模式。
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			if tcp, ok := conn.(*net.TCPConn); ok {
				_ = tcp.SetNoDelay(false)
			}
			return conn, nil
		},
	}
	conn, resp, err := dialer.DialContext(ctx, c.url, c.header)
	if err != nil {
		// 握手被网关以非 101 状态码拒绝时，gorilla 只返回 ErrBadHandshake（不带
		// 状态码）。这里把 HTTP 状态码并入错误串，供上层按 rust parse_ws_http_code
		// 的同款格式（"unexpected server response: 400"）识别登录码失效。
		if resp != nil && resp.StatusCode != 0 {
			return fmt.Errorf("protocol: dial: unexpected server response: %d: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("protocol: dial: %w", err)
	}
	c.conn = conn
	c.closed.Store(false)
	c.lastInboundAt = time.Now()

	readCtx, readCancel := context.WithCancel(context.Background())
	c.readCancel = readCancel
	go c.readLoop(readCtx)

	return nil
}

// StartHeartbeat is deprecated: game Heartbeat is owned by Session after Login
// (must include gid + client_version). Kept as no-op for API compatibility.
func (c *Client) StartHeartbeat() {}

// Close shuts down the connection and pending RPCs.
func (c *Client) Close() error {
	c.closed.Store(true)
	// 会话结束：丢弃未消费的一次性初始化凭据（对齐 rust end_session → clear）。
	c.tokens.Clear()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.hbCancel != nil {
		c.hbCancel()
		c.hbCancel = nil
	}
	if c.readCancel != nil {
		c.readCancel()
		c.readCancel = nil
	}
	for seq, req := range c.pending {
		c.settleLocked(req, rpcResult{err: fmt.Errorf("protocol: connection closed")})
		delete(c.pending, seq)
	}
	queue := c.queue
	c.queue = nil
	for _, req := range queue {
		c.settleLocked(req, rpcResult{err: fmt.Errorf("protocol: connection closed")})
	}
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// Send performs a request/response RPC wrapped in gatepb.Message. The request
// joins the five-class scheduler: Heartbeat/AntiData get critical lanes, the
// ambient ctx class (WithRequestClass) applies otherwise, defaulting to
// foreground.
func (c *Client) Send(ctx context.Context, service, method string, body []byte) ([]byte, *gatepb.Meta, error) {
	return c.send(ctx, service, method, body, true)
}

// SendNoReply writes a request without waiting for a reply, still passing
// through the same scheduler (bot sendMsgNoReply).
func (c *Client) SendNoReply(ctx context.Context, service, method string, body []byte) error {
	_, _, err := c.send(ctx, service, method, body, false)
	return err
}

func (c *Client) send(ctx context.Context, service, method string, body []byte, expectReply bool) ([]byte, *gatepb.Meta, error) {
	if c.closed.Load() {
		return nil, nil, fmt.Errorf("protocol: client closed")
	}
	class, lane := resolveRequestClass(method, "", "", AmbientRequestClass(ctx))

	c.mu.Lock()
	if c.conn == nil {
		c.mu.Unlock()
		return nil, nil, fmt.Errorf("连接未打开: %s", method)
	}
	// 每个班次有独立的排队配额：后台任务把自己的配额排满，也不会占掉前台/心跳的名额。
	queuedForClass := 0
	for _, q := range c.queue {
		if q != nil && !q.settled && q.class == class {
			queuedForClass++
		}
	}
	if queuedForClass >= maxQueuedByClass[class] {
		total := len(c.queue)
		pending := len(c.pending)
		c.mu.Unlock()
		return nil, nil, fmt.Errorf("请求等待队列已满: %s (class=%s, limit=%d, queued=%d, pending=%d)",
			method, class, maxQueuedByClass[class], total, pending)
	}
	req := &queuedRequest{
		service:     service,
		method:      method,
		body:        body,
		class:       class,
		lane:        lane,
		enqueuedAt:  time.Now(),
		ch:          make(chan rpcResult, 1),
		expectReply: expectReply,
		timeout:     defaultRequestTimeout,
	}
	c.queue = append(c.queue, req)
	c.drainQueueLocked()
	c.logRequestPressureLocked(time.Now())
	c.mu.Unlock()

	req.timer = time.AfterFunc(req.timeout, func() {
		stage := "queued"
		c.mu.Lock()
		if req.seq != 0 {
			stage = "pending"
		}
		c.mu.Unlock()
		c.settle(req, rpcResult{err: fmt.Errorf("请求超时: %s (stage=%s, pending=%d, queued=%d)",
			method, stage, c.PendingCount(), c.QueuedCount())})
	})
	if class == ClassBackground {
		// background 是「后台补数据」：拿不到空闲槽位就早点让路，而不是一路熬到请求超时。
		queueWaitMs := lowPriorityQueueWaitMs
		if req.timeout < time.Duration(queueWaitMs)*time.Millisecond {
			queueWaitMs = int(req.timeout / time.Millisecond)
		}
		req.queueWait = time.AfterFunc(time.Duration(queueWaitMs)*time.Millisecond, func() {
			c.mu.Lock()
			sent := req.seq != 0 || req.settled
			c.mu.Unlock()
			if sent {
				return
			}
			c.settle(req, rpcResult{err: &GatewayBusyError{Message: fmt.Sprintf(
				"网关繁忙，后台请求已让路: %s (waited=%dms, pending=%d, queued=%d)",
				method, queueWaitMs, c.PendingCount(), c.QueuedCount())}})
		})
	}

	select {
	case <-ctx.Done():
		c.settle(req, rpcResult{err: ctx.Err()})
		return nil, nil, ctx.Err()
	case res := <-req.ch:
		return res.body, res.meta, res.err
	}
}

// drainQueue dispatches as many queued requests as the class budgets allow.
func (c *Client) drainQueue() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.drainQueueLocked()
}

func (c *Client) drainQueueLocked() {
	for {
		// Drop settled entries still parked in the queue slice.
		live := c.queue[:0]
		for _, q := range c.queue {
			if q != nil && !q.settled {
				live = append(live, q)
			}
		}
		c.queue = live
		if len(c.queue) == 0 {
			return
		}

		idx := selectDispatchIndex(classCandidates(c.queue), classInFlights(c.pending), time.Now())
		if idx < 0 {
			return
		}
		req := c.queue[idx]
		c.queue = append(c.queue[:idx], c.queue[idx+1:]...)

		if c.conn == nil || c.closed.Load() {
			c.settleLocked(req, rpcResult{err: fmt.Errorf("连接未打开: %s", req.method)})
			continue
		}

		seq := c.clientSeq
		c.clientSeq++
		// 加密前登记在途请求，确保排队器能准确计算并发槽位。
		if req.expectReply {
			req.seq = seq
			c.pending[seq] = req
		}
		if err := c.writeFrameLocked(seq, req); err != nil {
			if req.expectReply {
				delete(c.pending, seq)
				req.seq = 0
			}
			c.settleLocked(req, rpcResult{err: fmt.Errorf("发送失败: %s: %w", req.method, err)})
			continue
		}
		if !req.expectReply {
			c.settleLocked(req, rpcResult{})
		}
	}
}

// writeFrameLocked encrypts+marshals+writes one frame. Caller holds c.mu.
func (c *Client) writeFrameLocked(seq int64, req *queuedRequest) error {
	finalBody := req.body
	if len(finalBody) > 0 && c.encryptor != nil {
		enc, err := c.encryptor.Encrypt(finalBody)
		if err != nil {
			return fmt.Errorf("protocol: encrypt: %w", err)
		}
		finalBody = enc
	}
	// 出站 token：有暂存的一次性 TSDK 初始化凭据则原子消费、恰好随本条
	// 消息发送一次（对齐 rust send_rpc 路径的 token_provider.next_marked）。
	token, staged := c.tokens.NextMarked()
	if staged {
		slog.Info("TSDK 初始化凭据已随本条请求发送",
			"service", req.service,
			"method", req.method,
		)
	}
	msg := &gatepb.Message{
		Meta: &gatepb.Meta{
			ServiceName: req.service,
			MethodName:  req.method,
			MessageType: int32(gatepb.MessageType_Request),
			ClientSeq:   seq,
			ServerSeq:   c.serverSeq,
		},
		Body:  finalBody,
		Token: token,
	}
	frame, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("protocol: marshal: %w", err)
	}
	if err := c.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		return fmt.Errorf("protocol: write: %w", err)
	}
	return nil
}

// settle finishes a request exactly once; safe from any goroutine.
func (c *Client) settle(req *queuedRequest, res rpcResult) {
	c.mu.Lock()
	c.settleLocked(req, res)
	c.mu.Unlock()
	c.drainQueue()
}

// settleLocked finishes a request exactly once; caller holds c.mu.
func (c *Client) settleLocked(req *queuedRequest, res rpcResult) {
	if req.settled {
		return
	}
	req.settled = true
	if req.seq != 0 {
		if _, ok := c.pending[req.seq]; ok {
			delete(c.pending, req.seq)
		}
		req.seq = 0
	}
	if req.timer != nil {
		req.timer.Stop()
		req.timer = nil
	}
	if req.queueWait != nil {
		req.queueWait.Stop()
		req.queueWait = nil
	}
	req.ch <- res
}

func (c *Client) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()
		if conn == nil {
			return
		}
		// No client read deadline — matches qq-farm-bot (idle detection is via Heartbeat silence).
		_, data, err := conn.ReadMessage()
		if err != nil {
			readErr := fmt.Errorf("protocol: read: %w", err)
			c.rejectAll(readErr)
			// Intentional Close sets closed before tearing down the conn.
			if !c.closed.Load() {
				c.mu.Lock()
				cb := c.onDisconnect
				c.mu.Unlock()
				if cb != nil {
					cb(readErr)
				}
			}
			return
		}
		c.handleFrame(data)
	}
}

func (c *Client) handleFrame(data []byte) {
	var msg gatepb.Message
	if err := proto.Unmarshal(data, &msg); err != nil {
		return
	}
	if msg.Meta == nil {
		return
	}
	c.mu.Lock()
	c.lastInboundAt = time.Now()
	if msg.Meta.ServerSeq > 0 && msg.Meta.ServerSeq > c.serverSeq {
		c.serverSeq = msg.Meta.ServerSeq
	}

	// Response/notify bodies are plaintext protobuf inside GateMessage
	// (qq-farm-bot network.ts: LoginReply.decode(msg.body) — no decrypt).
	// Only outbound request bodies are ACE-encrypted.
	body := msg.Body

	if msg.Meta.MessageType == int32(gatepb.MessageType_Response) {
		req, ok := c.pending[msg.Meta.ClientSeq]
		if ok {
			delete(c.pending, msg.Meta.ClientSeq)
		}
		c.mu.Unlock()
		if !ok {
			return
		}
		c.settleResponse(req, msg.Meta, body)
		return
	}

	if msg.Meta.MessageType == int32(gatepb.MessageType_Notify) {
		handler := c.onNotify
		c.mu.Unlock()
		if handler != nil {
			handler(msg.Meta.ServiceName, msg.Meta.MethodName, body)
		}
		return
	}

	// 其余帧（MessageType 缺失/未知）：对齐 rust gateway.rs dispatch_loop 的
	// 回包容错——部分大包如 FriendService.GetAll 不带标准 Response type。仅当
	// client_seq != 0 且命中 pending、method 名为空或与 pending 请求一致时，
	// 按回包容错完成；其余帧照旧丢弃（显式 Notify 一律走 onNotify，绝不当回包）。
	if msg.Meta.ClientSeq != 0 {
		if req, ok := c.pending[msg.Meta.ClientSeq]; ok &&
			(msg.Meta.MethodName == "" || msg.Meta.MethodName == req.method) {
			delete(c.pending, msg.Meta.ClientSeq)
			c.mu.Unlock()
			c.settleResponse(req, msg.Meta, body)
			return
		}
	}
	c.mu.Unlock()
}

// settleResponse 以回包语义完成一个在途请求：错误码非 0 记为失败，否则交付 body
// （对齐 rust handle_response：error_code != 0 → fail，否则 complete）。
func (c *Client) settleResponse(req *queuedRequest, meta *gatepb.Meta, body []byte) {
	if meta.ErrorCode != 0 {
		c.settle(req, rpcResult{meta: meta, err: fmt.Errorf("%s.%s error code=%d %s",
			meta.ServiceName, meta.MethodName, meta.ErrorCode, meta.ErrorMessage)})
		return
	}
	c.settle(req, rpcResult{body: body, meta: meta})
}

func (c *Client) rejectAll(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 读循环死亡 = 会话结束：清一次性初始化凭据（对齐 rust end_session → clear）。
	c.tokens.Clear()
	for seq, req := range c.pending {
		c.settleLocked(req, rpcResult{err: err})
		delete(c.pending, seq)
	}
	queue := c.queue
	c.queue = nil
	for _, req := range queue {
		c.settleLocked(req, rpcResult{err: err})
	}
}

// PendingCount / QueuedCount expose scheduler depth for logging.
func (c *Client) PendingCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.pending)
}

func (c *Client) QueuedCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.queue)
}

// GatewayLoad returns the low-priority gate snapshot (bot getGatewayLoad).
func (c *Client) GatewayLoad() gatewayLoad {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	load := gatewayLoad{
		pending: len(c.pending),
		queued:  len(c.queue),
	}
	for _, q := range c.queue {
		if q == nil || q.settled || q.class == ClassBackground {
			continue
		}
		load.blockingQueued++
	}
	var oldest int64
	for seq, req := range c.pending {
		_ = seq
		switch req.class {
		case ClassCritical:
			load.criticalPending++
		case ClassBackground:
			load.backgroundPending++
		default:
			load.businessPending++
			if req.class == ClassForeground {
				load.foregroundPending++
			}
		}
		if age := now.Sub(req.enqueuedAt).Milliseconds(); age > oldest {
			oldest = age
		}
	}
	load.heartbeatMisses = c.heartbeatMisses
	load.oldestPendingAgeMs = oldest
	return load
}

// IsGatewayIdleForBackground reports whether background traffic may fly now.
func (c *Client) IsGatewayIdleForBackground() bool {
	return isGatewayIdleForLowPriority(c.GatewayLoad())
}

// IsGatewayHealthyForBusiness reports whether farm/friend ticks may proceed.
func (c *Client) IsGatewayHealthyForBusiness() bool {
	return isGatewayHealthyForBusiness(c.GatewayLoad())
}

// NoteHeartbeatMiss / ClearHeartbeatMisses maintain the liveness counter used
// by the load snapshot (wired by the Session heartbeat loop).
func (c *Client) NoteHeartbeatMiss() {
	c.mu.Lock()
	c.heartbeatMisses++
	c.mu.Unlock()
}

func (c *Client) ClearHeartbeatMisses() {
	c.mu.Lock()
	c.heartbeatMisses = 0
	c.mu.Unlock()
}

// InboundSilenceMs returns time since the last inbound frame.
func (c *Client) InboundSilenceMs() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Since(c.lastInboundAt).Milliseconds()
}

// StageInitToken 暂存一次性 TSDK 初始化凭据（登录 bindUser 成功后由 Session
// 调用，对齐 rust gateway.rs login 第 6 步 stage_init_token）。
func (c *Client) StageInitToken(value string) (int, error) {
	return c.tokens.StageInitToken(value)
}

// ClearTokens 丢弃未消费的一次性初始化凭据（会话结束/重连时调用，
// 对齐 rust end_session → token_provider.clear()）。
func (c *Client) ClearTokens() { c.tokens.Clear() }

// ReplaceEncryptor 原子替换加密器（TSDK 重建时调用，对齐 rust
// Gateway.replace_encryptor）。新 encryptor 对后续所有出站帧立即可见；
// 替换瞬间的在途请求不受影响。
func (c *Client) ReplaceEncryptor(e Encryptor) {
	c.mu.Lock()
	c.encryptor = e
	c.mu.Unlock()
}

// BeginRebuild 进入 TSDK 重建期：期间心跳判死静默阈值放宽到 90s
// （对齐 rust Gateway.begin_rebuild，WorkerLoop 据此放宽 silence 阈值）。
func (c *Client) BeginRebuild() { c.rebuilding.Store(true) }

// EndRebuild 退出 TSDK 重建期（对齐 rust Gateway.end_rebuild）。
func (c *Client) EndRebuild() { c.rebuilding.Store(false) }

// IsRebuilding reports whether the TSDK rebuild is in progress.
func (c *Client) IsRebuilding() bool { return c.rebuilding.Load() }

// Connected reports whether the websocket is dialed and open.
// Session 层离线短路（ACE AntiData 等）用它 + 会话状态等价 rust 的
// `gateway.phase() == Online` 判断。
func (c *Client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil && !c.closed.Load()
}

// logRequestPressureLocked throttles queue pressure warnings to one per 5s;
// a queue holding only background requests is normal and never logged.
func (c *Client) logRequestPressureLocked(now time.Time) {
	blocking := 0
	for _, q := range c.queue {
		if q != nil && !q.settled && q.class != ClassBackground {
			blocking++
		}
	}
	if blocking == 0 {
		return
	}
	if now.Sub(c.lastPressureLogAt) < requestPressureLogIntervalMs {
		return
	}
	c.lastPressureLogAt = now
	describe := func() string {
		out := ""
		n := 0
		for _, q := range c.pending {
			if n >= 6 {
				break
			}
			n++
			if out != "" {
				out += ","
			}
			out += fmt.Sprintf("%s#%d:%dms", q.method, q.seq, now.Sub(q.enqueuedAt).Milliseconds())
		}
		if out == "" {
			out = "none"
		}
		return out
	}
	queuedDesc := ""
	n := 0
	for _, q := range c.queue {
		if q == nil || q.settled || n >= 8 {
			continue
		}
		n++
		if queuedDesc != "" {
			queuedDesc += ","
		}
		queuedDesc += q.method
	}
	if queuedDesc == "" {
		queuedDesc = "none"
	}
	pressureLog(fmt.Sprintf("Gateway 请求压力: pending=%d, queued=%d, active=%s, queuedMethods=%s",
		len(c.pending), len(c.queue), describe(), queuedDesc))
}

// pressureLog is a hook so tests can capture pressure warnings.
var pressureLog = func(msg string) {}

// --- adapter views for selectDispatchIndex ---

type queueView struct {
	req *queuedRequest
}

func (v queueView) requestClass() RequestClass { return v.req.class }
func (v queueView) lane() CriticalLane         { return v.req.lane }
func (v queueView) enqueuedAt() time.Time      { return v.req.enqueuedAt }

type inflightView struct {
	req *queuedRequest
}

func (v inflightView) requestClass() RequestClass { return v.req.class }
func (v inflightView) lane() CriticalLane         { return v.req.lane }

func classCandidates(queue []*queuedRequest) []classCandidate {
	out := make([]classCandidate, 0, len(queue))
	for _, q := range queue {
		out = append(out, queueView{req: q})
	}
	return out
}

func classInFlights(pending map[int64]*queuedRequest) []classInFlight {
	out := make([]classInFlight, 0, len(pending))
	for _, q := range pending {
		out = append(out, inflightView{req: q})
	}
	return out
}
