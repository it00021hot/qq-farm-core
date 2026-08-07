// Package protocol implements the QQ Farm WSS gateway client skeleton.
package protocol

import (
	"context"
	"fmt"
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

// Client is a WSS gateway client.
type Client struct {
	url            string
	header         http.Header
	encryptor      Encryptor
	heartbeatEvery time.Duration
	onNotify       NotifyHandler

	mu         sync.Mutex
	conn       *websocket.Conn
	clientSeq  int64
	serverSeq  int64
	pending    map[int64]chan rpcResult
	closed     atomic.Bool
	hbCancel   context.CancelFunc
	readCancel context.CancelFunc
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
		pending:        make(map[int64]chan rpcResult),
		clientSeq:      1,
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
	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second}
	conn, _, err := dialer.DialContext(ctx, c.url, c.header)
	if err != nil {
		return fmt.Errorf("protocol: dial: %w", err)
	}
	c.conn = conn
	c.closed.Store(false)

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
	for seq, ch := range c.pending {
		ch <- rpcResult{err: fmt.Errorf("protocol: connection closed")}
		close(ch)
		delete(c.pending, seq)
	}
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// Send performs a request/response RPC wrapped in gatepb.Message.
func (c *Client) Send(ctx context.Context, service, method string, body []byte) ([]byte, *gatepb.Meta, error) {
	if c.closed.Load() {
		return nil, nil, fmt.Errorf("protocol: client closed")
	}

	finalBody := body
	if len(finalBody) > 0 && c.encryptor != nil {
		enc, err := c.encryptor.Encrypt(finalBody)
		if err != nil {
			return nil, nil, fmt.Errorf("protocol: encrypt: %w", err)
		}
		finalBody = enc
	}

	c.mu.Lock()
	if c.conn == nil {
		c.mu.Unlock()
		return nil, nil, fmt.Errorf("protocol: not connected")
	}
	seq := c.clientSeq
	c.clientSeq++
	serverSeq := c.serverSeq
	ch := make(chan rpcResult, 1)
	c.pending[seq] = ch

	msg := &gatepb.Message{
		Meta: &gatepb.Meta{
			ServiceName: service,
			MethodName:  method,
			MessageType: int32(gatepb.MessageType_Request),
			ClientSeq:   seq,
			ServerSeq:   serverSeq,
		},
		Body:  finalBody,
		Token: CreateGatewayToken(),
	}
	frame, err := proto.Marshal(msg)
	if err != nil {
		delete(c.pending, seq)
		c.mu.Unlock()
		return nil, nil, fmt.Errorf("protocol: marshal: %w", err)
	}
	err = c.conn.WriteMessage(websocket.BinaryMessage, frame)
	c.mu.Unlock()
	if err != nil {
		c.mu.Lock()
		delete(c.pending, seq)
		c.mu.Unlock()
		return nil, nil, fmt.Errorf("protocol: write: %w", err)
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, seq)
		c.mu.Unlock()
		return nil, nil, ctx.Err()
	case res := <-ch:
		return res.body, res.meta, res.err
	}
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
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			c.rejectAll(fmt.Errorf("protocol: read: %w", err))
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
	if msg.Meta.ServerSeq > 0 {
		c.mu.Lock()
		if msg.Meta.ServerSeq > c.serverSeq {
			c.serverSeq = msg.Meta.ServerSeq
		}
		c.mu.Unlock()
	}

	// Response/notify bodies are plaintext protobuf inside GateMessage
	// (qq-farm-bot network.ts: LoginReply.decode(msg.body) — no decrypt).
	// Only outbound request bodies are ACE-encrypted.
	body := msg.Body

	if msg.Meta.MessageType == int32(gatepb.MessageType_Response) {
		c.mu.Lock()
		ch, ok := c.pending[msg.Meta.ClientSeq]
		if ok {
			delete(c.pending, msg.Meta.ClientSeq)
		}
		c.mu.Unlock()
		if !ok {
			return
		}
		if msg.Meta.ErrorCode != 0 {
			ch <- rpcResult{meta: msg.Meta, err: fmt.Errorf("%s.%s error code=%d %s",
				msg.Meta.ServiceName, msg.Meta.MethodName, msg.Meta.ErrorCode, msg.Meta.ErrorMessage)}
		} else {
			ch <- rpcResult{body: body, meta: msg.Meta}
		}
		close(ch)
		return
	}

	if msg.Meta.MessageType == int32(gatepb.MessageType_Notify) {
		c.mu.Lock()
		handler := c.onNotify
		c.mu.Unlock()
		if handler != nil {
			handler(msg.Meta.ServiceName, msg.Meta.MethodName, body)
		}
	}
}

func (c *Client) rejectAll(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for seq, ch := range c.pending {
		ch <- rpcResult{err: err}
		close(ch)
		delete(c.pending, seq)
	}
}
