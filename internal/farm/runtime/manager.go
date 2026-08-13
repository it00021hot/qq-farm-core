// Package runtime manages per-account farm sessions.
package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	urlquery "net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/activitycenter"
	"github.com/it00021hot/qq-farm-core/internal/farm/deviceprofile"
	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/hub"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/acepb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/activitypb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/friendpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/gatepb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/interactpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/itempb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/plantpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/seasonpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/userpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/protocol"
	"github.com/it00021hot/qq-farm-core/internal/farm/push"
	"github.com/it00021hot/qq-farm-core/internal/farm/stats"
	"github.com/it00021hot/qq-farm-core/internal/farm/tsdk"
	"github.com/it00021hot/qq-farm-core/internal/vars"
	"google.golang.org/protobuf/proto"
)

// AccountStatus is broadcast on the hub.
type AccountStatus string

const (
	StatusStopped  AccountStatus = "stopped"
	StatusStarting AccountStatus = "starting"
	StatusRunning  AccountStatus = "running"
	StatusStopping AccountStatus = "stopping"
	StatusError    AccountStatus = "error"
)

// SessionConfig is runtime config applied to a session.
type SessionConfig struct {
	AccountID     string
	Code          string // one-time gateway auth code
	Platform      string // qq | wx
	OSName        string // from login URL os=
	ClientVersion string // from login URL ver=
	GatewayURL    string
	WASMPath      string
	DataRoot      string
	ShareFile     string // optional share.txt path (farm.shareFile / FARM_SHARE_FILE)
	AccountConfig logic.AccountConfig
	DeviceModel   string
	SysSoftware   string
	FarmInterval  time.Duration // legacy fixed tick; 0 uses AccountConfig.Intervals jitter
	PushWebhook   string        // optional Bark/WeCom webhook for offline/error alerts
}

// Session is one running account worker.
type Session struct {
	id     string
	cfg    SessionConfig
	hub    *hub.Hub
	status AccountStatus

	mu                 sync.Mutex
	cancel             context.CancelFunc
	runCtx             context.Context
	tsdk               *tsdk.Runtime
	client             *protocol.Client
	gameAPI            *game.API
	lastErr            string
	gid                int64
	openID             string
	clientVer          string
	playerLevel        int64
	playerExp          int64
	gold               int64
	nick               string
	avatar             string
	landCount          int
	friendCount        int
	isFirstFarmCheck   bool
	farmOpMu           sync.Mutex
	lastPushAt         time.Time
	lastFertBuyCheckAt time.Time
	fertilizerGiftDate string
	dailyState         DailyState
	dailyDateKey       string
	// steal/help patrol visited GIDs (bubble+probe rotation); reset when probe pool exhausted
	stealPatrolVisited map[int64]struct{}
	helpPatrolVisited  map[int64]struct{}
	helpState          *friendHelpState
	// next scheduled tick times for status nextChecks countdowns
	nextFarmAt        time.Time
	nextStealAt       time.Time
	nextHelpAt        time.Time
	startedAt         time.Time
	baselineExp       int64
	baselineGold      int64
	sessionExpGained  int64
	sessionGoldGained int64
	lastStatusPushAt  time.Time
}

func newSession(cfg SessionConfig, h *hub.Hub) *Session {
	accountID := parseAccountID(cfg.AccountID)
	return &Session{
		id:                 cfg.AccountID,
		cfg:                cfg,
		hub:                h,
		status:             StatusStopped,
		isFirstFarmCheck:   true,
		stealPatrolVisited: make(map[int64]struct{}),
		helpPatrolVisited:  make(map[int64]struct{}),
		helpState:          newFriendHelpState(accountID, cfg.DataRoot),
	}
}

func (s *Session) setStatus(st AccountStatus, detail string) {
	s.mu.Lock()
	prev := s.status
	s.status = st
	if detail != "" {
		s.lastErr = detail
	}
	webhook := s.cfg.PushWebhook
	accountID := s.id
	s.mu.Unlock()
	if s.hub != nil {
		s.hub.PublishJSON("account_status", parseAccountID(accountID), map[string]any{
			"status": string(st),
			"detail": detail,
		})
	}
	// Push offline/error alerts once when leaving a healthy/running state.
	if webhook != "" && st == StatusError && prev != StatusError {
		title := fmt.Sprintf("农场账号 %s 异常", accountID)
		body := detail
		if body == "" {
			body = string(st)
		}
		go func() { _ = push.Notify(webhook, title, body) }()
	}
}

// Status returns the current status.
func (s *Session) Status() AccountStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// ApplyConfig updates account automation/strategy config.
func (s *Session) ApplyConfig(cfg logic.AccountConfig) {
	s.mu.Lock()
	s.cfg.AccountConfig = cfg
	s.mu.Unlock()
	if s.hub != nil {
		s.hub.PublishJSON("account_config", parseAccountID(s.id), cfg)
	}
}

// Start dials the gateway, performs Login, then runs the farm loop.
// Returns only after login succeeds or fails (does not fake "running").
func (s *Session) Start(parent context.Context) error {
	s.mu.Lock()
	if s.status == StatusRunning || s.status == StatusStarting {
		s.mu.Unlock()
		return fmt.Errorf("session %s already running", s.id)
	}
	s.mu.Unlock()

	s.setStatus(StatusStarting, "")
	ctx, cancel := context.WithCancel(parent)
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()

	ready := make(chan error, 1)
	go s.run(ctx, ready)

	select {
	case err := <-ready:
		if err != nil {
			// run() already marked StatusError + will remove itself from manager
			persistRunStatus(parseAccountID(s.id), RunError, false)
			return err
		}
		persistRunStatus(parseAccountID(s.id), RunRunning, true)
		return nil
	case <-time.After(30 * time.Second):
		cancel()
		s.setStatus(StatusError, "登录超时")
		persistRunStatus(parseAccountID(s.id), RunError, false)
		m := getManager()
		if m != nil {
			m.mu.Lock()
			if cur, ok := m.sess[s.id]; ok && cur == s {
				delete(m.sess, s.id)
			}
			m.mu.Unlock()
		}
		return fmt.Errorf("登录超时：网关未在 30s 内完成连接/登录")
	}
}

// Stop cancels the session.
func (s *Session) Stop() {
	s.setStatus(StatusStopping, "正在停止")
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Session) run(ctx context.Context, ready chan<- error) {
	defer func() {
		s.cleanup()
		st := s.Status()
		if st != StatusError {
			s.setStatus(StatusStopped, "已停止")
			persistRunStatus(parseAccountID(s.id), RunStopped, false)
		}
		m := getManager()
		if m != nil {
			m.mu.Lock()
			if cur, ok := m.sess[s.id]; ok && cur == s {
				delete(m.sess, s.id)
			}
			m.mu.Unlock()
		}
	}()

	dev := deviceprofile.Resolve(s.cfg.OSName)
	if s.cfg.DeviceModel == "" {
		s.cfg.DeviceModel = dev.DeviceID
	}
	if s.cfg.SysSoftware == "" {
		s.cfg.SysSoftware = dev.SysSoftware
	}
	if s.cfg.OSName == "" {
		s.cfg.OSName = dev.OS
	}

	// TSDK is mandatory: encrypt Login/Heartbeat/ACE like the Node bot.
	if s.cfg.WASMPath == "" {
		ready <- fmt.Errorf("未配置 farm.wasmPath，无法安全登录")
		return
	}
	rt, err := tsdk.New(tsdk.Config{
		WASMPath:    s.cfg.WASMPath,
		AccountID:   s.id,
		DataRoot:    s.cfg.DataRoot,
		DeviceModel: s.cfg.DeviceModel,
		OSName:      s.cfg.OSName,
		SysSoftware: s.cfg.SysSoftware,
	})
	if err != nil {
		ready <- fmt.Errorf("TSDK 初始化失败: %w", err)
		return
	}
	if err := rt.Init(ctx); err != nil {
		rt.Destroy()
		ready <- fmt.Errorf("TSDK 初始化失败: %w", err)
		return
	}
	s.mu.Lock()
	s.tsdk = rt
	s.mu.Unlock()

	url := s.cfg.GatewayURL
	if url == "" {
		url = "wss://gate-obt.nqf.qq.com/prod/ws"
	}
	platform := s.cfg.Platform
	if platform == "" {
		platform = "qq"
	}
	osName := s.cfg.OSName
	if osName == "" {
		osName = "Windows"
	}
	ver := s.cfg.ClientVersion
	if ver == "" {
		ver = vars.Config.GetString("farm.clientVersion")
	}
	if ver == "" {
		ver = "1.13.0.5_20260723"
	}
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	url = url + sep + "platform=" + urlquery.QueryEscape(platform) +
		"&os=" + urlquery.QueryEscape(osName) +
		"&ver=" + urlquery.QueryEscape(ver) +
		"&code=" + urlquery.QueryEscape(s.cfg.Code)

	var enc protocol.Encryptor
	s.mu.Lock()
	rt = s.tsdk
	s.mu.Unlock()
	if rt == nil {
		ready <- fmt.Errorf("TSDK 未就绪，拒绝明文登录（避免异常协议流量）")
		return
	}
	enc = rt

	header := make(map[string][]string)
	header["Origin"] = []string{deviceprofile.DefaultGatewayOrigin}
	header["User-Agent"] = []string{dev.UserAgent}

	client := protocol.NewClient(protocol.Options{
		URL:            url,
		Header:         header,
		Encryptor:      enc,
		HeartbeatEvery: 25 * time.Second,
		OnNotify:       s.handleNotify,
		OnDisconnect:   s.failTransport,
	})

	slog.Info("farm gateway dial",
		"account", s.id,
		"platform", platform,
		"os", osName,
		"ver", ver,
	)

	dialCtx, dialCancel := context.WithTimeout(ctx, 15*time.Second)
	err = client.Connect(dialCtx)
	dialCancel()
	if err != nil {
		_ = client.Close()
		s.setStatus(StatusError, err.Error())
		ready <- fmt.Errorf("网关连接失败（请检查 Code 是否有效）: %w", err)
		return
	}

	s.mu.Lock()
	s.client = client
	s.gameAPI = &game.API{Sender: client}
	s.clientVer = ver
	s.runCtx = ctx
	s.mu.Unlock()

	if err := s.doLogin(ctx, client, ver, dev); err != nil {
		_ = client.Close()
		s.setStatus(StatusError, err.Error())
		ready <- err
		return
	}

	s.setStatus(StatusRunning, "")
	s.mu.Lock()
	s.startedAt = time.Now()
	s.mu.Unlock()
	ready <- nil

	// Post-login loops (aligned with qq-farm-bot): game Heartbeat + ACE.
	go s.gameHeartbeatLoop(ctx)
	go s.aceLoop(ctx)
	go s.friendLoop(ctx, "steal")
	go s.friendLoop(ctx, "help")
	go s.friendLoop(ctx, "bad")
	go s.runAcceptFriendsBootstrap(ctx)
	go s.runFriendRefreshBootstrap(ctx)
	go s.dailyLoop(ctx)
	go s.runDailyBootstrap(ctx)

	for {
		delay := s.nextFarmDelay()
		s.setNextCheckAt("farm", time.Now().Add(delay))
		s.publishStatusSnapshot()
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			s.farmTick(ctx)
		}
	}
}

func (s *Session) doLogin(ctx context.Context, client *protocol.Client, ver string, dev deviceprofile.Profile) error {
	if ver == "" {
		ver = "1.13.0.5_20260723"
	}
	// Exact LoginRequest shape from network.ts sendLogin — sparse DeviceInfo only.
	sysSoft := dev.SysSoftware
	if sysSoft == "" {
		sysSoft = "Windows"
	}
	req := &userpb.LoginRequest{
		SceneId: "1234567",
		DeviceInfo: &userpb.DeviceInfo{
			ClientVersion: ver,
			SysSoftware:   sysSoft,
			ScreenWidth:   0,
		},
		ReportData: &userpb.ReportData{
			MinigameChannel: "other-qq",
			MinigamePlatid:  2,
		},
	}
	loginCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	requestBody, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("登录请求编码失败: %w", err)
	}
	body, _, err := client.Send(loginCtx, "gamepb.userpb.UserService", "Login", requestBody)
	if err != nil {
		return fmt.Errorf("登录失败: %w", err)
	}
	var reply userpb.LoginReply
	if err := proto.Unmarshal(body, &reply); err != nil {
		return fmt.Errorf("登录响应解析失败: %w", err)
	}
	if reply.Basic == nil || reply.Basic.Gid == 0 {
		return fmt.Errorf("登录失败: 响应缺少账号信息（Code 可能已失效）")
	}
	if reply.GetTimeNowMillis() > 0 {
		logic.SyncServerTime(reply.GetTimeNowMillis())
	}

	s.mu.Lock()
	s.gid = reply.Basic.Gid
	s.openID = reply.Basic.OpenId
	s.playerLevel = reply.Basic.Level
	s.playerExp = reply.Basic.Exp
	s.gold = reply.Basic.Gold
	s.baselineExp = reply.Basic.Exp
	s.baselineGold = reply.Basic.Gold
	s.sessionExpGained = 0
	s.sessionGoldGained = 0
	s.nick = reply.Basic.Name
	s.avatar = reply.Basic.AvatarUrl
	rt := s.tsdk
	if s.gameAPI != nil {
		s.gameAPI.GID = reply.Basic.Gid
	}
	s.mu.Unlock()

	// bindUser(openId) BEFORE heartbeat / ACE (network.ts login success order).
	if rt != nil && reply.Basic.OpenId != "" {
		if err := rt.BindUser(reply.Basic.OpenId); err != nil {
			slog.Warn("tsdk bindUser failed", "account", s.id, "err", err)
		}
	}

	slog.Info("farm login ok",
		"account", s.id,
		"gid", reply.Basic.Gid,
		"name", reply.Basic.Name,
		"level", reply.Basic.Level,
	)
	if reply.Basic.Name != "" {
		persistAccountProfile(parseAccountID(s.id), reply.Basic)
	}
	s.publishStatusSnapshotThrottled(true)
	s.postLoginHooks(ctx)
	return nil
}

// postLoginHooks mirrors Node network.ts / invite.ts after-login side effects.
func (s *Session) postLoginHooks(ctx context.Context) {
	s.mu.Lock()
	api := s.gameAPI
	platform := s.cfg.Platform
	dataRoot := s.cfg.DataRoot
	shareFile := s.cfg.ShareFile
	s.mu.Unlock()
	if api == nil {
		return
	}
	if platform == "" {
		platform = "qq"
	}

	go func() {
		bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if _, err := api.GetUserSettings(bg); err != nil {
			slog.Debug("get user settings failed", "account", s.id, "err", err)
		}
	}()

	// WX invite: Node invite.ts reads share.txt and ReportArkClick (scene_id=1256).
	if platform == "wx" {
		go func() {
			bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
			defer cancel()
			processInviteCodes(bg, api, dataRoot, shareFile)
		}()
	}
}

// gameHeartbeatLoop mirrors qq-farm-bot startHeartbeat:
// UserService.Heartbeat every 25s; kill only after >30s without a heartbeat reply.
func (s *Session) gameHeartbeatLoop(ctx context.Context) {
	const (
		heartbeatInterval   = 25 * time.Second
		heartbeatSilence    = 30 * time.Second
		heartbeatRPCTimeout = 20 * time.Second
		maxHeartbeatMiss    = 1
	)
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	lastResponse := time.Now()
	missCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			client := s.client
			gid := s.gid
			ver := s.clientVer
			s.mu.Unlock()
			if client == nil || gid == 0 {
				continue
			}
			if ver == "" {
				ver = "1.13.0.5_20260723"
			}

			silence := time.Since(lastResponse)
			if silence > heartbeatSilence {
				missCount++
				slog.Warn("farm heartbeat silence",
					"account", s.id,
					"silence_sec", int(silence.Seconds()),
					"miss", missCount,
				)
				if missCount >= maxHeartbeatMiss {
					s.failTransport(fmt.Errorf("protocol: connection closed: heartbeat timeout (%ds no response)", int(silence.Seconds())))
					return
				}
			}

			heartbeatBody, marshalErr := proto.Marshal(&userpb.HeartbeatRequest{Gid: gid, ClientVersion: ver})
			if marshalErr != nil {
				continue
			}
			hbCtx, cancel := context.WithTimeout(ctx, heartbeatRPCTimeout)
			raw, _, err := client.Send(hbCtx, "gamepb.userpb.UserService", "Heartbeat", heartbeatBody)
			cancel()
			if err != nil {
				// Node: Heartbeat send/reply errors are swallowed; only silence kills the session.
				slog.Warn("farm heartbeat failed", "account", s.id, "err", err)
				continue
			}

			lastResponse = time.Now()
			missCount = 0
			var reply userpb.HeartbeatReply
			if proto.Unmarshal(raw, &reply) == nil && reply.GetServerTime() > 0 {
				logic.SyncServerTime(normalizeServerTimeMs(reply.GetServerTime()))
			}
		}
	}
}

// normalizeServerTimeMs mirrors bot toTimeSec inverse for SyncServerTime (expects ms).
func normalizeServerTimeMs(v int64) int64 {
	if v <= 0 {
		return 0
	}
	if v > 1e12 {
		return v
	}
	return v * 1000
}

// aceLoop mirrors qq-farm-bot startAceRuntime timers.
func (s *Session) aceLoop(ctx context.Context) {
	anti := time.NewTicker(5 * time.Second)
	proc := time.NewTicker(5 * time.Second)
	tick := time.NewTicker(25 * time.Second)
	speed := time.NewTicker(30 * time.Second)
	status := time.NewTicker(150 * time.Second)
	defer anti.Stop()
	defer proc.Stop()
	defer tick.Stop()
	defer speed.Stop()
	defer status.Stop()

	lastSpeed := time.Now()
	var antiBusy sync.Mutex

	for {
		select {
		case <-ctx.Done():
			return
		case <-anti.C:
			if !antiBusy.TryLock() {
				continue
			}
			s.sendAntiData(ctx)
			antiBusy.Unlock()
		case <-proc.C:
			s.mu.Lock()
			rt := s.tsdk
			s.mu.Unlock()
			if rt != nil {
				_ = rt.ProcessReceivedData()
			}
		case <-tick.C:
			s.mu.Lock()
			rt := s.tsdk
			s.mu.Unlock()
			if rt != nil {
				_ = rt.HeartbeatTick()
			}
		case <-speed.C:
			s.mu.Lock()
			rt := s.tsdk
			s.mu.Unlock()
			if rt != nil {
				elapsed := time.Since(lastSpeed).Milliseconds()
				_ = rt.DetectSpeedHack(elapsed)
				lastSpeed = time.Now()
			}
		case <-status.C:
			s.mu.Lock()
			rt := s.tsdk
			s.mu.Unlock()
			if rt != nil {
				_ = rt.SendStatus()
			}
		}
	}
}

func (s *Session) sendAntiData(ctx context.Context) {
	s.mu.Lock()
	rt := s.tsdk
	client := s.client
	s.mu.Unlock()
	if rt == nil || client == nil {
		return
	}
	data, err := rt.GetDataToServer()
	if err != nil || len(data) == 0 {
		return
	}
	req := &acepb.AntiDataRequest{Data: data}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	requestBody, marshalErr := proto.Marshal(req)
	if marshalErr != nil {
		cancel()
		return
	}
	body, _, err := client.Send(cctx, "gamepb.acepb.AceService", "AntiData", requestBody)
	cancel()
	if err != nil {
		slog.Warn("ACE AntiData failed", "account", s.id, "err", err)
		return
	}
	var reply acepb.AntiDataReply
	if err := proto.Unmarshal(body, &reply); err != nil {
		return
	}
	if len(reply.Result) > 0 {
		_ = rt.SendDataFromServer(reply.Result)
	}
}

// RunFarmOp executes an own-farm operation for this session.
func (s *Session) RunFarmOp(ctx context.Context, op string) (hadWork bool, actions []string, err error) {
	s.farmOpMu.Lock()
	defer s.farmOpMu.Unlock()

	s.mu.Lock()
	api := s.gameAPI
	cfg := s.cfg.AccountConfig
	playerLevel := s.playerLevel
	gold := s.gold
	s.mu.Unlock()
	if api == nil {
		return false, nil, fmt.Errorf("farm session is not connected")
	}
	hadWork, actions, lands, err := RunFarmOperation(
		ctx, api, cfg, op,
		WithStatsAccount(parseAccountID(s.id)),
		WithPlayerState(playerLevel, gold),
		WithOperationLimitsSink(s.ensureHelpState().updateLimits),
	)
	s.mu.Lock()
	s.isFirstFarmCheck = false
	if len(lands) > 0 {
		s.landCount = len(lands)
	}
	s.mu.Unlock()
	if s.hub != nil {
		msg := strings.Join(actions, "/")
		if msg == "" {
			if op == "all" {
				msg = "巡查完成"
			} else if op != "" {
				msg = op
			} else {
				msg = "操作完成"
			}
		}
		if msg == "all" {
			msg = "巡查完成"
		}
		tag := "农场"
		if err != nil {
			tag = "错误"
			if len(actions) == 0 {
				msg = err.Error()
			} else {
				msg = msg + " · " + err.Error()
			}
		}
		s.hub.PublishJSON("farm_operation", parseAccountID(s.id), map[string]any{
			"op":      op,
			"hadWork": hadWork,
			"actions": actions,
			"error":   errorText(err),
			"tag":     tag,
			"event":   "农场操作",
			"message": msg,
			"isWarn":  err != nil,
		})
	}
	if err != nil {
		s.failTransport(err)
	}
	return hadWork, actions, err
}

// GetLands returns the current own-farm lands for this session.
func (s *Session) GetLands(ctx context.Context) ([]logic.LandInfo, error) {
	s.mu.Lock()
	api := s.gameAPI
	s.mu.Unlock()
	if api == nil {
		return nil, fmt.Errorf("farm session is not connected")
	}
	lands, reply, err := api.AllLands(ctx)
	if err == nil {
		s.setLandCount(len(lands))
		if reply != nil {
			s.ensureHelpState().updateLimits(reply.OperationLimits)
		}
	}
	return lands, err
}

// GetBagItems returns flat bag items for this session.
func (s *Session) GetBagItems(ctx context.Context) ([]corepb.Item, error) {
	s.mu.Lock()
	api := s.gameAPI
	s.mu.Unlock()
	if api == nil {
		return nil, fmt.Errorf("farm session is not connected")
	}
	reply, err := api.Bag(ctx)
	if err != nil {
		return nil, err
	}
	return game.GetBagItems(reply), nil
}

// BagSeeds returns the seed entries currently in the bag for this session.
// Unlike the shop catalog, every seed with a Plant.json entry is included, so
// activity/legacy seeds that are not for sale are still visible.
func (s *Session) BagSeeds(ctx context.Context) ([]logic.BagSeed, error) {
	s.mu.Lock()
	api := s.gameAPI
	s.mu.Unlock()
	if api == nil {
		return nil, fmt.Errorf("farm session is not connected")
	}
	return api.BagSeeds(ctx)
}

// GetAvailableSeeds returns shop seeds with lock/sold-out flags for this account (bot getAvailableSeeds).
// Falls back to local catalog when shop RPC fails or is unavailable.
func (s *Session) GetAvailableSeeds(ctx context.Context) ([]logic.AvailableShopSeed, error) {
	s.mu.Lock()
	api := s.gameAPI
	playerLevel := s.playerLevel
	s.mu.Unlock()

	list := make([]logic.AvailableShopSeed, 0)
	if api != nil {
		shop, err := api.ShopInfo(ctx, 2)
		if err == nil && shop != nil {
			for _, goods := range shop.GoodsList {
				if goods == nil || goods.ItemId <= 0 {
					continue
				}
				requiredLevel := int64(0)
				for _, cond := range goods.Conds {
					if cond.Type == 1 && cond.Param > requiredLevel {
						requiredLevel = cond.Param
					}
				}
				soldOut := goods.LimitCount > 0 && goods.BoughtNum >= goods.LimitCount
				price := goods.Price
				reqLv := requiredLevel
				list = append(list, logic.AvailableShopSeed{
					SeedID:        goods.ItemId,
					GoodsID:       goods.Id,
					Name:          logic.GetPlantNameBySeedID(goods.ItemId),
					Price:         &price,
					RequiredLevel: &reqLv,
					Size:          logic.GetPlantSizeBySeedID(goods.ItemId),
					Locked:        !goods.Unlocked || (playerLevel > 0 && requiredLevel > playerLevel),
					SoldOut:       soldOut,
				})
			}
		}
	}

	if len(list) == 0 {
		return logic.CatalogAvailableSeeds(playerLevel), nil
	}

	sort.SliceStable(list, func(i, j int) bool {
		av, bv := int64(9999), int64(9999)
		if list[i].RequiredLevel != nil {
			av = *list[i].RequiredLevel
		}
		if list[j].RequiredLevel != nil {
			bv = *list[j].RequiredLevel
		}
		if av != bv {
			return av < bv
		}
		return list[i].SeedID < list[j].SeedID
	})
	return list, nil
}

// SellBagItems sells items from the bag.
func (s *Session) SellBagItems(ctx context.Context, items []corepb.Item) error {
	s.mu.Lock()
	api := s.gameAPI
	s.mu.Unlock()
	if api == nil {
		return fmt.Errorf("farm session is not connected")
	}
	_, err := api.Sell(ctx, items)
	return err
}

// UseBagItem uses one bag item via ItemService.Use, falling back to BatchUse.
func (s *Session) UseBagItem(ctx context.Context, itemID, count int64) error {
	s.mu.Lock()
	api := s.gameAPI
	s.mu.Unlock()
	if api == nil {
		return fmt.Errorf("farm session is not connected")
	}
	if itemID <= 0 || count <= 0 {
		return fmt.Errorf("invalid item")
	}
	if _, useErr := api.Use(ctx, itemID, count); useErr == nil {
		return nil
	} else {
		bag, bagErr := api.Bag(ctx)
		if bagErr != nil {
			return useErr
		}
		items := []corepb.Item{}
		for _, it := range game.GetBagItems(bag) {
			if it.Id != itemID {
				continue
			}
			items = append(items, it)
		}
		if len(items) == 0 {
			return useErr
		}
		if _, batchErr := api.BatchUse(ctx, items); batchErr != nil {
			return useErr
		}
		return nil
	}
}

// Friends returns the live game friend list for this session.
func (s *Session) Friends(ctx context.Context) ([]friendpb.GameFriend, error) {
	s.farmOpMu.Lock()
	defer s.farmOpMu.Unlock()

	s.mu.Lock()
	api := s.gameAPI
	cfg := s.cfg.AccountConfig
	s.mu.Unlock()
	if api == nil {
		return nil, fmt.Errorf("farm session is not connected")
	}
	friends, err := loadFriends(ctx, s, api, cfg)
	if err == nil {
		s.setFriendCount(len(friends))
	}
	return friends, err
}

// InteractRecords returns live visitor interaction records (bot Analytics visitors tab).
func (s *Session) InteractRecords(ctx context.Context) (*interactpb.InteractRecordsReply, error) {
	s.farmOpMu.Lock()
	defer s.farmOpMu.Unlock()

	s.mu.Lock()
	api := s.gameAPI
	s.mu.Unlock()
	if api == nil {
		return nil, fmt.Errorf("farm session is not connected")
	}
	return api.InteractRecords(ctx)
}

// SyncFriends refreshes the known friend GID cache for this account.
func (s *Session) SyncFriends(ctx context.Context) error {
	s.farmOpMu.Lock()
	defer s.farmOpMu.Unlock()

	s.mu.Lock()
	api := s.gameAPI
	cfg := s.cfg.AccountConfig
	s.mu.Unlock()
	if api == nil {
		return fmt.Errorf("farm session is not connected")
	}
	friends, err := loadFriends(ctx, s, api, cfg)
	if err != nil {
		return err
	}
	ReplaceFriendsToDB(parseAccountID(s.id), s.GID(), friends)
	return nil
}

// FriendOp performs one manual friend interaction.
func (s *Session) FriendOp(ctx context.Context, gid int64, op string) error {
	if gid <= 0 {
		return fmt.Errorf("friend gid is required")
	}
	s.farmOpMu.Lock()
	defer s.farmOpMu.Unlock()

	s.mu.Lock()
	api := s.gameAPI
	cfg := s.cfg.AccountConfig
	myGID := s.gid
	s.mu.Unlock()
	if api == nil {
		return fmt.Errorf("farm session is not connected")
	}
	if myGID > 0 && gid == myGID {
		return fmt.Errorf("不能对自己执行好友操作")
	}

	var (
		outcome friendVisitOutcome
		err     error
	)
	switch op {
	case "steal":
		outcome, err = stealFriend(ctx, s, api, cfg, myGID, gid)
		if outcome.Count > 0 {
			accountID := parseAccountID(s.id)
			stats.RecordOp(accountID, 0, "steal", outcome.Count)
			if cfg.Automation.Sell {
				sold, sellErr := sellAllFruits(ctx, api)
				if sellErr != nil && err == nil {
					err = fmt.Errorf("steal ok but sell failed: %w", sellErr)
				} else if sold > 0 {
					stats.RecordOp(accountID, 0, "sell", 1)
				}
			}
		}
		if outcome.HelpCount > 0 {
			stats.RecordOp(parseAccountID(s.id), 0, "help", outcome.HelpCount)
		}
	case "help", "water", "weed", "bug":
		var limitReached bool
		outcome, limitReached, err = manualHelpFriend(ctx, s, api, gid, op)
		_ = limitReached
		if outcome.Count > 0 {
			stats.RecordOp(parseAccountID(s.id), 0, "help", outcome.Count)
		}
	case "bad":
		outcome, err = badFriend(ctx, s, api, myGID, gid)
	default:
		return fmt.Errorf("unsupported friend operation %q", op)
	}
	result := "ok"
	if err != nil {
		result = "error"
	}
	detail := map[string]any{
		"count":       outcome.Count,
		"plants":      outcome.Plants,
		"summary":     outcome.Summary,
		"helpSummary": outcome.HelpSummary,
		"weed":        outcome.Weed,
		"bug":         outcome.Bug,
		"water":       outcome.Water,
		"putBug":      outcome.PutBug,
		"putWeed":     outcome.PutWeed,
		"error":       errorText(err),
	}
	writeInteractLog(parseAccountID(s.id), 0, gid, op, result, detail)
	return err
}

// FriendLands returns a friend's current farm state and always leaves the visit.
func (s *Session) FriendLands(ctx context.Context, gid int64) ([]logic.LandInfo, error) {
	if gid <= 0 {
		return nil, fmt.Errorf("friend gid is required")
	}
	s.farmOpMu.Lock()
	defer s.farmOpMu.Unlock()

	s.mu.Lock()
	api := s.gameAPI
	myGID := s.gid
	s.mu.Unlock()
	if api == nil {
		return nil, fmt.Errorf("farm session is not connected")
	}
	if myGID > 0 && gid == myGID {
		return nil, fmt.Errorf("不能查看自己的好友农场，请使用个人农场")
	}
	lands, err := api.VisitEnter(ctx, gid, enterReasonFriend)
	if err != nil {
		if handleFriendEnterError(s, gid, err) {
			return nil, fmt.Errorf("friend blacklisted after enter error: %w", err)
		}
		return nil, err
	}
	if leaveErr := api.VisitLeave(ctx, gid); leaveErr != nil {
		return nil, leaveErr
	}
	return lands, nil
}

// Config returns a snapshot of this session's account config.
func (s *Session) Config() logic.AccountConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.AccountConfig
}

// GameAPI returns the API bound to this connected session.
func (s *Session) GameAPI() *game.API {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gameAPI
}

// GID returns the logged-in game identifier.
func (s *Session) GID() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gid
}

// NextChecksSnapshot mirrors bot status_sync nextChecks countdowns (seconds remaining).
type NextChecksSnapshot struct {
	FarmRemainSec   int `json:"farmRemainSec"`
	FriendRemainSec int `json:"friendRemainSec"`
	HelpRemainSec   int `json:"helpRemainSec"`
	StealRemainSec  int `json:"stealRemainSec"`
}

// StatusSnapshot is the live panel payload for dashboard / status API.
type StatusSnapshot struct {
	AccountID         uint64                 `json:"accountId"`
	RunStatus         uint8                  `json:"runStatus"`
	Online            bool                   `json:"online"`
	Level             int64                  `json:"level"`
	Exp               int64                  `json:"exp"`
	Gold              int64                  `json:"gold"`
	Nick              string                 `json:"nick"`
	Avatar            string                 `json:"avatar"`
	LandCount         int                    `json:"landCount"`
	FriendCount       int                    `json:"friendCount"`
	LastError         string                 `json:"lastError"`
	GID               int64                  `json:"gid"`
	UpdatedAt         int64                  `json:"updatedAt"`
	Uptime            int64                  `json:"uptime"`
	SessionExpGained  int64                  `json:"sessionExpGained"`
	SessionGoldGained int64                  `json:"sessionGoldGained"`
	LevelProgress     logic.LevelExpProgress `json:"levelProgress"`
	Operations        map[string]int         `json:"operations"`
	NextChecks        NextChecksSnapshot     `json:"nextChecks"`
}

// remainSecUntil matches bot syncStatus Math.ceil((nextRunAt - nowMs) / 1000).
func remainSecUntil(at time.Time) int {
	if at.IsZero() {
		return 0
	}
	return remainSecFromDuration(time.Until(at))
}

func remainSecFromDuration(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int((d + time.Second - 1) / time.Second)
}

func (s *Session) setNextCheckAt(kind string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch kind {
	case "farm":
		s.nextFarmAt = at
	case "steal":
		s.nextStealAt = at
	case "help":
		s.nextHelpAt = at
	}
}

// Snapshot returns live runtime fields for the UI.
func (s *Session) Snapshot() StatusSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	run := RunStopped
	online := false
	switch s.status {
	case StatusRunning, StatusStarting:
		run = RunRunning
		online = s.status == StatusRunning
	case StatusError:
		run = RunError
	case StatusStopping:
		run = RunRunning
	}
	farmRemain := remainSecUntil(s.nextFarmAt)
	helpRemain := remainSecUntil(s.nextHelpAt)
	stealRemain := remainSecUntil(s.nextStealAt)
	friendRemain := helpRemain
	if stealRemain > friendRemain {
		friendRemain = stealRemain
	}
	uptime := int64(0)
	if !s.startedAt.IsZero() && online {
		uptime = int64(time.Since(s.startedAt).Seconds())
		if uptime < 0 {
			uptime = 0
		}
	}
	accountID := parseAccountID(s.id)
	ops := stats.TodayOperations(accountID, 0)
	dayExp, dayGold := stats.TodayExpGold(accountID, 0)
	sessionExp := s.sessionExpGained
	if dayExp > sessionExp {
		sessionExp = dayExp
	}
	sessionGold := s.sessionGoldGained
	if dayGold > sessionGold {
		sessionGold = dayGold
	}
	return StatusSnapshot{
		AccountID:         accountID,
		RunStatus:         run,
		Online:            online,
		Level:             s.playerLevel,
		Exp:               s.playerExp,
		Gold:              s.gold,
		Nick:              s.nick,
		Avatar:            s.avatar,
		LandCount:         s.landCount,
		FriendCount:       s.friendCount,
		LastError:         s.lastErr,
		GID:               s.gid,
		UpdatedAt:         time.Now().Unix(),
		Uptime:            uptime,
		SessionExpGained:  sessionExp,
		SessionGoldGained: sessionGold,
		LevelProgress:     logic.GetLevelExpProgress(s.playerLevel, s.playerExp),
		Operations:        ops,
		NextChecks: NextChecksSnapshot{
			FarmRemainSec:   farmRemain,
			FriendRemainSec: friendRemain,
			HelpRemainSec:   helpRemain,
			StealRemainSec:  stealRemain,
		},
	}
}

func (s *Session) publishStatusSnapshot() {
	s.publishStatusSnapshotThrottled(false)
}

func (s *Session) publishStatusSnapshotThrottled(force bool) {
	if s.hub == nil {
		return
	}
	s.mu.Lock()
	if !force && !s.lastStatusPushAt.IsZero() && time.Since(s.lastStatusPushAt) < 1500*time.Millisecond {
		s.mu.Unlock()
		return
	}
	s.lastStatusPushAt = time.Now()
	s.mu.Unlock()
	snap := s.Snapshot()
	s.hub.PublishJSON("status", snap.AccountID, snap)
}

func (s *Session) setLandCount(n int) {
	s.mu.Lock()
	s.landCount = n
	s.mu.Unlock()
}

func (s *Session) setFriendCount(n int) {
	s.mu.Lock()
	s.friendCount = n
	s.mu.Unlock()
}

// SetFriendCount updates cached friend count from external callers (status API).
func (s *Session) SetFriendCount(n int) { s.setFriendCount(n) }

// SetLandCount updates cached land count from external callers.
func (s *Session) SetLandCount(n int) { s.setLandCount(n) }

func (s *Session) handleNotify(service, method string, body []byte) {
	if len(body) == 0 {
		return
	}

	var event gatepb.EventMessage
	eventBody := body
	messageType := ""
	if err := proto.Unmarshal(body, &event); err == nil && event.MessageType != "" {
		messageType = event.MessageType
		eventBody = event.Body
	}
	if len(eventBody) == 0 && !strings.Contains(messageType, "Kickout") {
		return
	}

	// Protocol sniffing: log every push so unknown activity/season pushes are
	// visible before their payloads are understood. Info level so the sniffing
	// deployment sees pushes without enabling debug logging.
	slog.Info("farm push",
		"account", s.id,
		"service", service,
		"method", method,
		"messageType", messageType,
		"bodyLen", len(eventBody),
	)

	// P0: gate kickout — stop session to avoid zombies (Node: gatepb.KickoutNotify).
	if strings.Contains(messageType, "Kickout") {
		reason := "未知"
		var notify gatepb.KickoutNotify
		if err := proto.Unmarshal(eventBody, &notify); err == nil {
			if notify.ReasonMessage != "" {
				reason = notify.ReasonMessage
			}
		}
		slog.Warn("farm kickout", "account", s.id, "reason", reason, "type", messageType)
		s.setStatus(StatusError, "被踢下线: "+reason)
		persistRunStatus(parseAccountID(s.id), RunError, false)
		s.mu.Lock()
		cancel := s.cancel
		s.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		return
	}

	// P1: auto-accept friend applications pushed by the gate.
	if strings.Contains(messageType, "FriendApplicationReceivedNotify") {
		var notify friendpb.FriendApplicationReceivedNotify
		if err := proto.Unmarshal(eventBody, &notify); err != nil {
			return
		}
		gids := make([]int64, 0, len(notify.Applications))
		for _, app := range notify.Applications {
			if app != nil && app.Gid > 0 {
				gids = append(gids, app.Gid)
			}
		}
		if len(gids) == 0 {
			return
		}
		s.mu.Lock()
		api := s.gameAPI
		runCtx := s.runCtx
		s.mu.Unlock()
		if api == nil || runCtx == nil {
			return
		}
		go func() {
			if _, err := api.AcceptFriends(runCtx, gids); err != nil {
				slog.Debug("accept friend application notify failed", "account", s.id, "err", err)
				return
			}
			slog.Info("accepted friend applications from notify", "account", s.id, "count", len(gids))
		}()
		return
	}

	if strings.Contains(messageType, "ItemNotify") || (service == "gamepb.itempb.ItemService" && method == "ItemNotify") {
		s.applyItemNotify(eventBody)
		return
	}
	if strings.Contains(messageType, "BasicNotify") || (service == "gamepb.userpb.UserService" && method == "BasicNotify") {
		s.applyBasicNotify(eventBody)
		return
	}
	if strings.Contains(messageType, "BattlePassChangeNotify") ||
		(service == "gamepb.seasonpb.SeasonService" && method == "BattlePassChangeNotify") {
		s.applyBattlePassNotify(eventBody)
		return
	}
	if strings.Contains(messageType, "ActiviesChangeNotify") ||
		strings.Contains(messageType, "ActivityChangeNotify") ||
		(service == "gamepb.activitypb.ActivityService" && strings.Contains(method, "Activity")) {
		s.applyActivityChangeNotify(eventBody)
		return
	}
	if !strings.Contains(messageType, "LandsNotify") &&
		!(service == "gamepb.plantpb.PlantService" && method == "LandsNotify") {
		return
	}

	var notify plantpb.LandsNotify
	if err := proto.Unmarshal(eventBody, &notify); err != nil {
		return
	}
	if len(notify.Lands) == 0 {
		return
	}

	s.mu.Lock()
	myGID := s.gid
	cfg := s.cfg.AccountConfig
	runCtx := s.runCtx
	now := time.Now()
	if now.Sub(s.lastPushAt) < 500*time.Millisecond {
		s.mu.Unlock()
		return
	}
	if !cfg.Automation.FarmPush || !cfg.Automation.Farm {
		s.mu.Unlock()
		return
	}
	hostGID := notify.HostGid
	if hostGID != 0 && myGID > 0 && hostGID != myGID {
		s.mu.Unlock()
		return
	}
	if logic.InQuietHours(cfg.FriendQuietHours.Enabled, cfg.FriendQuietHours.Start, cfg.FriendQuietHours.End, now.Format("15:04")) {
		s.mu.Unlock()
		return
	}
	s.lastPushAt = now
	s.mu.Unlock()

	if runCtx == nil {
		return
	}
	go func() {
		if _, _, err := s.RunFarmOp(runCtx, "all"); err != nil {
			slog.Warn("farm push tick failed", "account", s.id, "err", err)
		}
	}()
}

// failTransport stops the account when the gateway socket is dead.
// Login Code is one-shot; reconnecting with the same code is useless.
func (s *Session) failTransport(err error) {
	if s == nil || err == nil || !isFatalTransportError(err) {
		return
	}
	if s.isQuiescing() {
		return
	}
	detail := "网关连接已断开（登录 Code 为一次性，无法自动重连）"
	slog.Warn("farm transport dead, stopping session", "account", s.id, "err", err)
	s.setStatus(StatusError, detail)
	persistRunStatus(parseAccountID(s.id), RunError, false)
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Session) applyItemNotify(body []byte) {
	var notify itempb.ItemNotify
	if err := proto.Unmarshal(body, &notify); err != nil {
		return
	}
	var expDelta, goldDelta int64
	s.mu.Lock()
	changed := false
	for _, chg := range notify.Items {
		if chg == nil || chg.Item == nil {
			continue
		}
		id := chg.Item.Id
		count := chg.Item.Count
		delta := chg.Delta
		switch id {
		case 1101: // exp
			if count > 0 {
				if s.playerExp != count {
					s.playerExp = count
					changed = true
				}
			} else if delta != 0 {
				next := s.playerExp + delta
				if next < 0 {
					next = 0
				}
				if s.playerExp != next {
					s.playerExp = next
					changed = true
				}
			}
			if delta > 0 {
				s.sessionExpGained += delta
				expDelta += delta
				changed = true
			}
		case 1, 1001: // gold
			if count > 0 {
				if s.gold != count {
					s.gold = count
					changed = true
				}
			} else if delta != 0 {
				next := s.gold + delta
				if next < 0 {
					next = 0
				}
				if s.gold != next {
					s.gold = next
					changed = true
				}
			}
			if delta > 0 {
				s.sessionGoldGained += delta
				goldDelta += delta
				changed = true
			}
		}
	}
	s.mu.Unlock()
	if expDelta > 0 || goldDelta > 0 {
		stats.RecordExpGold(parseAccountID(s.id), 0, expDelta, goldDelta)
	}
	if changed {
		s.publishStatusSnapshotThrottled(false)
	}
}

func (s *Session) applyBattlePassNotify(body []byte) {
	var notify seasonpb.BattlePassChangeNotify
	if err := proto.Unmarshal(body, &notify); err != nil {
		slog.Debug("BattlePassChangeNotify decode failed", "account", s.id, "err", err)
		return
	}
	pass := activitycenter.ApplySeasonPassNotify(notify.Pass)
	if pass == nil {
		return
	}
	title, _ := pass["title"].(string)
	if title == "" {
		title = "游记"
	}
	slog.Info("battle pass changed",
		"account", s.id,
		"title", title,
		"level", pass["level"],
		"progress", pass["progress"],
		"progressMax", pass["progressMax"],
	)
}

func (s *Session) applyActivityChangeNotify(body []byte) {
	var notify activitypb.ActiviesChangeNotify
	if err := proto.Unmarshal(body, &notify); err != nil {
		slog.Debug("ActiviesChangeNotify decode failed", "account", s.id, "err", err)
		return
	}
	if len(notify.Activities) == 0 {
		return
	}
	items := make([]logic.ActivityRegistryItem, 0, len(notify.Activities))
	for _, act := range notify.Activities {
		if act == nil {
			continue
		}
		items = append(items, logic.ActivityRegistryItem{
			ActivityID: strconv.FormatInt(act.ActivityId, 10),
			Type:       act.Type,
			Name:       strings.TrimSpace(act.Name),
			BeginTime:  act.BeginTime,
			EndTime:    act.EndTime,
		})
		slog.Info("activity change notify",
			"account", s.id,
			"activityId", act.ActivityId,
			"groupId", act.GroupId,
			"type", act.Type,
			"name", strings.TrimSpace(act.Name),
			"beginTime", act.BeginTime,
			"endTime", act.EndTime,
		)
	}
	logic.RegisterActivities(items)
}

func (s *Session) applyBasicNotify(body []byte) {
	var notify userpb.BasicNotify
	if err := proto.Unmarshal(body, &notify); err != nil || notify.Basic == nil {
		return
	}
	leveledUp := false
	s.mu.Lock()
	changed := false
	if notify.Basic.Level > 0 && s.playerLevel != notify.Basic.Level {
		s.playerLevel = notify.Basic.Level
		leveledUp = true
		changed = true
	}
	if notify.Basic.Exp > 0 && s.playerExp != notify.Basic.Exp {
		s.playerExp = notify.Basic.Exp
		changed = true
	}
	if notify.Basic.Gold > 0 && s.gold != notify.Basic.Gold {
		s.gold = notify.Basic.Gold
		changed = true
	}
	if notify.Basic.Name != "" && s.nick != notify.Basic.Name {
		s.nick = notify.Basic.Name
		changed = true
	}
	if notify.Basic.AvatarUrl != "" && s.avatar != notify.Basic.AvatarUrl {
		s.avatar = notify.Basic.AvatarUrl
		changed = true
	}
	s.mu.Unlock()
	if leveledUp {
		stats.RecordOp(parseAccountID(s.id), 0, "levelUp", 1)
	}
	if changed {
		s.publishStatusSnapshotThrottled(false)
	}
}

func (s *Session) playerExpSnapshot() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.playerExp
}

func (s *Session) ensureHelpState() *friendHelpState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.helpState == nil {
		accountID := parseAccountID(s.id)
		s.helpState = newFriendHelpState(accountID, s.cfg.DataRoot)
	}
	return s.helpState
}

// farmTick performs automatic own-farm operations (bot checkFarm).
// Friend quiet hours also suppress own-farm ticks, matching bot scheduler.checkFarm.
func (s *Session) farmTick(ctx context.Context) {
	cfg := s.Config()
	s.runBagMaintenance(ctx, cfg)
	s.maybeRunDailyOnDateChange(ctx, cfg)
	s.runTaskClaimsTick(ctx, cfg)
	if !cfg.Automation.Farm {
		return
	}
	if logic.InQuietHours(cfg.FriendQuietHours.Enabled, cfg.FriendQuietHours.Start, cfg.FriendQuietHours.End, time.Now().Format("15:04")) {
		return
	}
	hadWork, actions, err := s.RunFarmOp(ctx, "all")
	if err != nil {
		slog.Warn("farm tick failed", "account", s.id, "err", err)
	}
	if s.hub != nil && (hadWork || err != nil) {
		msg := strings.Join(actions, "/")
		if msg == "" {
			msg = "巡查完成"
		}
		tag := "农场"
		if err != nil {
			tag = "错误"
			if msg == "巡查完成" {
				msg = err.Error()
			} else {
				msg = msg + " · " + err.Error()
			}
		}
		s.hub.PublishJSON("farm_tick", parseAccountID(s.id), map[string]any{
			"hadWork": hadWork,
			"actions": actions,
			"error":   errorText(err),
			"tag":     tag,
			"event":   "农场巡查",
			"message": msg,
			"isWarn":  err != nil,
		})
	}
	s.publishStatusSnapshot()
}

// friendLoop keeps friend work independent from farm ticks while serializing game RPCs.
func (s *Session) friendLoop(ctx context.Context, kind string) {
	for {
		cfg := s.Config()
		enabled := cfg.Automation.Friend &&
			((kind == "steal" && cfg.Automation.FriendSteal) ||
				(kind == "help" && cfg.Automation.FriendHelp) ||
				(kind == "bad" && cfg.Automation.FriendBad))
		if enabled {
			s.runFriendTick(ctx, kind)
		}
		delay := 30 * time.Second
		if enabled {
			delay = s.nextFriendDelay(cfg, kind)
		}
		s.setNextCheckAt(kind, time.Now().Add(delay))
		// Only push status when this loop is actively working; idle loops were spamming
		// WS status every ~30s and making the overview countdown flicker.
		if enabled {
			s.publishStatusSnapshot()
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (s *Session) runFriendTick(ctx context.Context, kind string) {
	if shouldAbortFriendPatrol(ctx, s) {
		return
	}
	cfg := s.Config()
	if logic.InQuietHours(cfg.FriendQuietHours.Enabled, cfg.FriendQuietHours.Start, cfg.FriendQuietHours.End, time.Now().Format("15:04")) {
		return
	}
	s.farmOpMu.Lock()
	defer s.farmOpMu.Unlock()
	s.mu.Lock()
	api := s.gameAPI
	s.mu.Unlock()
	if api == nil {
		return
	}
	// Auto-accept pending friend applications (bot: independent of friend automation toggle).
	if n, acceptErr := AcceptPendingFriends(ctx, api); acceptErr != nil {
		slog.Debug("accept friends failed", "account", s.id, "err", acceptErr)
	} else if n > 0 {
		slog.Info("accepted friend applications", "account", s.id, "count", n)
	}
	var err error
	switch kind {
	case "steal":
		if s.stealPatrolVisited == nil {
			s.stealPatrolVisited = make(map[int64]struct{})
		}
		_, err = RunStealTick(ctx, s, s.stealPatrolVisited)
	case "help":
		if s.helpPatrolVisited == nil {
			s.helpPatrolVisited = make(map[int64]struct{})
		}
		_, err = RunHelpTick(ctx, s, s.helpPatrolVisited)
	case "bad":
		_, err = RunBadOnce(ctx, s)
	}
	if err != nil {
		slog.Warn("friend tick failed", "account", s.id, "kind", kind, "err", err)
	}
}

// runAcceptFriendsBootstrap mirrors bot: accept pending applications shortly after login.
func (s *Session) runAcceptFriendsBootstrap(ctx context.Context) {
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}
	s.farmOpMu.Lock()
	defer s.farmOpMu.Unlock()
	s.mu.Lock()
	api := s.gameAPI
	s.mu.Unlock()
	if api == nil {
		return
	}
	if n, err := AcceptPendingFriends(ctx, api); err != nil {
		slog.Debug("accept friends bootstrap failed", "account", s.id, "err", err)
	} else if n > 0 {
		slog.Info("accepted friend applications on bootstrap", "account", s.id, "count", n)
	}
}

// runFriendRefreshBootstrap fully replaces the persisted friend list shortly after
// login so re-login never leaves stale friend rows in the DB.
func (s *Session) runFriendRefreshBootstrap(ctx context.Context) {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}
	s.farmOpMu.Lock()
	defer s.farmOpMu.Unlock()
	s.mu.Lock()
	api := s.gameAPI
	cfg := s.cfg.AccountConfig
	myGID := s.gid
	s.mu.Unlock()
	if api == nil {
		return
	}
	friends, err := loadFriends(ctx, s, api, cfg)
	if err != nil {
		slog.Warn("friend refresh bootstrap failed", "account", s.id, "err", err)
		return
	}
	ReplaceFriendsToDB(parseAccountID(s.id), myGID, friends)
	s.setFriendCount(len(friends))
	slog.Info("friend list refreshed on login", "account", s.id, "count", len(friends))
}

func (s *Session) nextFriendDelay(cfg logic.AccountConfig, kind string) time.Duration {
	min, max := cfg.Intervals.StealMin, cfg.Intervals.StealMax
	switch kind {
	case "help":
		min, max = cfg.Intervals.HelpMin, cfg.Intervals.HelpMax
	case "bad":
		// Bad visits up to 20 friends; use help interval so it is less frequent than steal.
		min, max = cfg.Intervals.HelpMin, cfg.Intervals.HelpMax
	}
	if min <= 0 {
		min = 10
	}
	if max < min {
		max = min
	}
	return time.Duration(min+rand.IntN(max-min+1)) * time.Second
}

func (s *Session) runBagMaintenance(ctx context.Context, cfg logic.AccountConfig) {
	now := time.Now()
	interval := time.Duration(cfg.FertilizerBuyCheckIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = time.Hour
	}

	s.mu.Lock()
	api := s.gameAPI
	shouldBuy := (cfg.Automation.FertilizerBuyOrganic || cfg.Automation.FertilizerBuyNormal) &&
		(now.Sub(s.lastFertBuyCheckAt) >= interval)
	// Bot opens fertilizer gifts every farm tick while enabled (no day-done gate).
	shouldOpenGifts := cfg.Automation.FertilizerGift
	if shouldBuy {
		s.lastFertBuyCheckAt = now
	}
	s.mu.Unlock()
	if api == nil || (!shouldBuy && !shouldOpenGifts) {
		return
	}

	bag, err := api.Bag(ctx)
	if err != nil {
		slog.Warn("farm bag maintenance failed", "account", s.id, "err", err)
		return
	}
	items := game.GetBagItems(bag)
	if shouldOpenGifts {
		if opened, err := openFertilizerGiftPacks(ctx, api, items); err != nil {
			slog.Warn("farm fertilizer gift opening failed", "account", s.id, "err", err)
		} else if opened > 0 {
			slog.Info("farm fertilizer gifts opened", "account", s.id, "count", opened)
		}
	}
	if !shouldBuy {
		return
	}
	normalHours, organicHours := fertilizerContainerHours(items)
	if cfg.Automation.FertilizerBuyOrganic &&
		cfg.FertilizerBuyOrganicCount > 0 &&
		organicHours < float64(cfg.FertilizerBuyOrganicThresholdHours) {
		if _, err := api.Purchase(ctx, game.OrganicMallGoodsID, int32(cfg.FertilizerBuyOrganicCount)); err != nil {
			slog.Warn("farm organic fertilizer purchase failed", "account", s.id, "err", err)
		}
	}
	if cfg.Automation.FertilizerBuyNormal &&
		cfg.FertilizerBuyNormalCount > 0 &&
		normalHours < float64(cfg.FertilizerBuyNormalThresholdHours) {
		if _, err := api.Purchase(ctx, game.InorganicMallGoodsID, int32(cfg.FertilizerBuyNormalCount)); err != nil {
			slog.Warn("farm normal fertilizer purchase failed", "account", s.id, "err", err)
		}
	}
}

func fertilizerContainerHours(items []corepb.Item) (normal, organic float64) {
	for _, item := range items {
		switch item.Id {
		case game.NormalFertilizerID:
			normal = float64(item.Count) / 3600
		case game.OrganicFertilizerID:
			organic = float64(item.Count) / 3600
		}
	}
	return normal, organic
}

const fertilizerContainerLimitHours = 990.0

var (
	normalFertItemHours = map[int64]float64{
		80001: 1, 80002: 4, 80003: 8, 80004: 12,
	}
	organicFertItemHours = map[int64]float64{
		80011: 1, 80012: 4, 80013: 8, 80014: 12,
	}
	fertilizerRelatedIDs = map[int64]struct{}{
		100003: {}, 100004: {},
		80001: {}, 80002: {}, 80003: {}, 80004: {},
		80011: {}, 80012: {}, 80013: {}, 80014: {},
	}
)

func isFertilizerRelatedItem(itemID int64) bool {
	if itemID <= 0 || itemID == game.NormalFertilizerID || itemID == game.OrganicFertilizerID {
		return false
	}
	if _, ok := fertilizerRelatedIDs[itemID]; ok {
		return true
	}
	info := logic.GetItemByID(itemID)
	if info == nil {
		return false
	}
	it := strings.ToLower(info.InteractionType)
	return it == "fertilizer" || it == "fertilizerpro"
}

func fertilizerItemTypeAndHours(itemID int64) (kind string, perItemHours float64) {
	if h, ok := normalFertItemHours[itemID]; ok {
		return "normal", h
	}
	if h, ok := organicFertItemHours[itemID]; ok {
		return "organic", h
	}
	info := logic.GetItemByID(itemID)
	if info == nil {
		return "other", 0
	}
	switch strings.ToLower(info.InteractionType) {
	case "fertilizer":
		return "normal", 1
	case "fertilizerpro":
		return "organic", 1
	default:
		return "other", 0
	}
}

// openFertilizerGiftPacks mirrors bot warehouse.autoOpenFertilizerGiftPacks.
func openFertilizerGiftPacks(ctx context.Context, api *game.API, items []corepb.Item) (opened int, err error) {
	merged := map[int64]int64{}
	for _, item := range items {
		if !isFertilizerRelatedItem(item.Id) || item.Count <= 0 {
			continue
		}
		merged[item.Id] += item.Count
	}
	if len(merged) == 0 {
		return 0, nil
	}
	normalHours, organicHours := fertilizerContainerHours(items)
	for id, count := range merged {
		kind, perItemHours := fertilizerItemTypeAndHours(id)
		useCount := count
		if kind == "normal" || kind == "organic" {
			current := normalHours
			if kind == "organic" {
				current = organicHours
			}
			if current >= fertilizerContainerLimitHours {
				continue
			}
			if perItemHours > 0 {
				remain := fertilizerContainerLimitHours - current
				maxByHours := int64(remain / perItemHours)
				if maxByHours < useCount {
					useCount = maxByHours
				}
				if useCount <= 0 {
					continue
				}
			}
		}
		if _, useErr := api.BatchUse(ctx, []corepb.Item{{Id: id, Count: useCount}}); useErr != nil {
			continue
		}
		opened += int(useCount)
		if kind == "normal" && perItemHours > 0 {
			normalHours += float64(useCount) * perItemHours
		}
		if kind == "organic" && perItemHours > 0 {
			organicHours += float64(useCount) * perItemHours
		}
		_ = waitFarmDelay(ctx, 100*time.Millisecond)
	}
	return opened, nil
}

func (s *Session) runDailyBootstrap(ctx context.Context) {
	time.Sleep(4 * time.Second)
	s.farmOpMu.Lock()
	defer s.farmOpMu.Unlock()
	s.mu.Lock()
	api := s.gameAPI
	cfg := s.cfg.AccountConfig
	accountID := parseAccountID(s.id)
	s.mu.Unlock()
	if api == nil {
		return
	}
	seedActivityRegistry(ctx, api)
	RunDailyRoutines(ctx, api, cfg, accountID, &s.dailyState, true)
	RunTaskClaims(ctx, api, cfg, accountID, &s.dailyState)
}

// seedActivityRegistry refreshes the activity registry from GetSeasonInfo so
// activity-restricted fruit selling has real schedules (instead of unknown).
func seedActivityRegistry(ctx context.Context, api *game.API) {
	seedCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if _, err := api.GetSeasonInfo(seedCtx); err != nil {
		slog.Debug("seed activity registry failed", "err", err)
	}
}

func (s *Session) dailyLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
	case <-ticker.C:
		cfg := s.Config()
		s.maybeRunDailyOnDateChange(ctx, cfg)
		s.farmOpMu.Lock()
		s.mu.Lock()
		api := s.gameAPI
		accountID := parseAccountID(s.id)
		s.mu.Unlock()
		if api != nil {
			RunDailyRoutines(ctx, api, cfg, accountID, &s.dailyState, false)
		}
		s.farmOpMu.Unlock()
		}
	}
}

func (s *Session) maybeRunDailyOnDateChange(ctx context.Context, cfg logic.AccountConfig) {
	today := time.Now().Format("2006-01-02")
	s.mu.Lock()
	if s.dailyDateKey == today {
		s.mu.Unlock()
		return
	}
	s.dailyDateKey = today
	api := s.gameAPI
	s.mu.Unlock()
	if api == nil {
		return
	}
	s.farmOpMu.Lock()
	RunDailyRoutines(ctx, api, cfg, parseAccountID(s.id), &s.dailyState, true)
	s.farmOpMu.Unlock()
}

func (s *Session) runTaskClaimsTick(ctx context.Context, cfg logic.AccountConfig) {
	if !cfg.Automation.Task {
		return
	}
	s.farmOpMu.Lock()
	defer s.farmOpMu.Unlock()
	s.mu.Lock()
	api := s.gameAPI
	accountID := parseAccountID(s.id)
	s.mu.Unlock()
	if api == nil {
		return
	}
	RunTaskClaims(ctx, api, cfg, accountID, &s.dailyState)
}

func (s *Session) nextFarmDelay() time.Duration {
	s.mu.Lock()
	cfg := s.cfg
	s.mu.Unlock()
	if cfg.FarmInterval > 0 {
		return cfg.FarmInterval
	}
	min, max := cfg.AccountConfig.Intervals.FarmMin, cfg.AccountConfig.Intervals.FarmMax
	if min <= 0 {
		min = 20
	}
	if max < min {
		max = min
	}
	return time.Duration(min+rand.IntN(max-min+1)) * time.Second
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (s *Session) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil {
		_ = s.client.Close()
		s.client = nil
	}
	if s.tsdk != nil {
		s.tsdk.Destroy()
		s.tsdk = nil
	}
	s.cancel = nil
}

// AccountManager owns sessions keyed by account ID.
type AccountManager struct {
	hub  *hub.Hub
	mu   sync.Mutex
	sess map[string]*Session
}

// NewAccountManager creates a manager that broadcasts via h.
func NewAccountManager(h *hub.Hub) *AccountManager {
	if h == nil {
		h = hub.New()
	}
	return &AccountManager{hub: h, sess: make(map[string]*Session)}
}

// Hub returns the status broadcast hub.
func (m *AccountManager) Hub() *hub.Hub { return m.hub }

// StartAccount starts (or restarts) an account session.
func (m *AccountManager) StartAccount(ctx context.Context, cfg SessionConfig) error {
	if cfg.AccountID == "" {
		return fmt.Errorf("account id required")
	}
	if strings.TrimSpace(cfg.Code) == "" {
		return fmt.Errorf("连接缺少一次性 Code")
	}
	if cfg.Platform == "" {
		cfg.Platform = "qq"
	}
	// Never wipe a loaded AccountConfig just because plantingStrategy was omitted;
	// fill missing strategy only.
	if cfg.AccountConfig.PlantingStrategy == "" {
		cfg.AccountConfig.PlantingStrategy = logic.StrategyPreferred
	}
	if cfg.AccountConfig.BagSeedFallbackStrategy == "" {
		cfg.AccountConfig.BagSeedFallbackStrategy = logic.StrategyLevel
	}
	if cfg.AccountConfig.Automation.Fertilizer == "" {
		cfg.AccountConfig.Automation.Fertilizer = logic.FertilizerSmart
	}
	if cfg.GatewayURL == "" {
		cfg.GatewayURL = vars.Config.GetString("farm.gatewayUrl")
	}
	if cfg.WASMPath == "" {
		cfg.WASMPath = vars.Config.GetString("farm.wasmPath")
	}
	if cfg.DataRoot == "" {
		cfg.DataRoot = vars.Config.GetString("farm.tsdkDataDir")
	}
	if cfg.ShareFile == "" {
		cfg.ShareFile = vars.Config.GetString("farm.shareFile")
	}
	if cfg.PushWebhook == "" {
		cfg.PushWebhook = vars.Config.GetString("farm.pushWebhook")
	}

	m.mu.Lock()
	if old, ok := m.sess[cfg.AccountID]; ok {
		old.Stop()
		delete(m.sess, cfg.AccountID)
	}
	s := newSession(cfg, m.hub)
	m.sess[cfg.AccountID] = s
	m.mu.Unlock()

	return s.Start(ctx)
}

// StopAccount stops a running account.
func (m *AccountManager) StopAccount(accountID string) error {
	m.mu.Lock()
	s, ok := m.sess[accountID]
	if ok {
		delete(m.sess, accountID)
	}
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("account %s not found", accountID)
	}
	s.Stop()
	return nil
}

// ApplyConfig applies config to a live session (or stores for next start if absent).
func (m *AccountManager) ApplyConfig(accountID string, cfg logic.AccountConfig) error {
	m.mu.Lock()
	s, ok := m.sess[accountID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("account %s not found", accountID)
	}
	s.ApplyConfig(cfg)
	return nil
}

// Status returns account status or stopped if unknown.
func (m *AccountManager) Status(accountID string) AccountStatus {
	m.mu.Lock()
	s, ok := m.sess[accountID]
	m.mu.Unlock()
	if !ok {
		return StatusStopped
	}
	return s.Status()
}

// Session returns the connected session for an account.
func (m *AccountManager) Session(accountID string) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sess[accountID]
	return s, ok
}

// StopAll stops every tracked account session.
func (m *AccountManager) StopAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.sess))
	for id := range m.sess {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		_ = m.StopAccount(id)
	}
}

// List returns account IDs currently tracked.
func (m *AccountManager) List() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.sess))
	for id := range m.sess {
		out = append(out, id)
	}
	return out
}

func parseAccountID(id string) uint64 {
	var n uint64
	for _, c := range id {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + uint64(c-'0')
	}
	return n
}
