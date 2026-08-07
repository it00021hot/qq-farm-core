// Package tsdk loads and drives the QQ Farm TSDK WASM module via wazero.
//
// Host imports a.a–a.v mirror Node's tsdk-runtime.ts degradation behavior.
// Init order is documented in constants.go and docs/tsdk-runtime.md.
package tsdk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// Config controls Runtime construction.
type Config struct {
	// WASMPath is the path to tsdk.wasm. Required.
	WASMPath string
	// AccountID scopes the per-account data directory.
	AccountID string
	// DataRoot is the parent of tsdk/<accountId>/. Defaults to ./data.
	DataRoot string
	// DeviceModel / OSName / SysSoftware feed import a.j.
	DeviceModel string
	OSName      string
	SysSoftware string
}

// Runtime wraps a wazero TSDK instance for one account.
type Runtime struct {
	cfg     Config
	dataDir string

	deviceModel string
	osName      string
	sysSoftware string

	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	runtime   wazero.Runtime
	mod       api.Module
	ready     bool
	destroyed bool
	userBound bool
	host      *host
}

// New creates an uninitialized Runtime. Call Init before Encrypt/Decrypt.
func New(cfg Config) (*Runtime, error) {
	if cfg.WASMPath == "" {
		return nil, fmt.Errorf("tsdk: WASMPath required")
	}
	if cfg.AccountID == "" {
		cfg.AccountID = "default"
	}
	if cfg.DataRoot == "" {
		cfg.DataRoot = "data"
	}
	dataDir := filepath.Join(cfg.DataRoot, "tsdk", cfg.AccountID)
	rt := &Runtime{
		cfg:         cfg,
		dataDir:     dataDir,
		deviceModel: cfg.DeviceModel,
		osName:      cfg.OSName,
		sysSoftware: cfg.SysSoftware,
	}
	rt.host = newHost(rt)
	return rt, nil
}

// DataDir returns the per-account writable TSDK data directory.
func (r *Runtime) DataDir() string { return r.dataDir }

// Init loads wasm, registers host imports, decrypts mergewasm segments, and calls G().
func (r *Runtime) Init(parent context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ready {
		return nil
	}
	if r.destroyed {
		return fmt.Errorf("tsdk: runtime destroyed")
	}

	wasm, err := os.ReadFile(r.cfg.WASMPath)
	if err != nil {
		return fmt.Errorf("tsdk: read wasm: %w", err)
	}
	if err := VerifyWASMHash(wasm); err != nil {
		return err
	}
	if err := os.MkdirAll(r.dataDir, 0o755); err != nil {
		return fmt.Errorf("tsdk: mkdir data: %w", err)
	}

	ctx, cancel := context.WithCancel(parent)
	r.ctx = ctx
	r.cancel = cancel
	r.runtime = wazero.NewRuntime(ctx)

	if err := r.instantiateHost(ctx); err != nil {
		r.cleanupLocked()
		return err
	}

	mod, err := r.runtime.Instantiate(ctx, wasm)
	if err != nil {
		r.cleanupLocked()
		return fmt.Errorf("tsdk: instantiate: %w", err)
	}
	r.mod = mod

	required := []string{"x", "y", "A", "B", "E", "G", "H", "M", "N", "O", "P", "aa", "ba", "ca", "fa"}
	for _, name := range required {
		if mod.ExportedFunction(name) == nil {
			r.cleanupLocked()
			return fmt.Errorf("tsdk: missing export %s", name)
		}
	}
	if mod.Memory() == nil {
		r.cleanupLocked()
		return fmt.Errorf("tsdk: missing memory export w")
	}

	decrypt := mod.ExportedFunction("__mergewasm_shared____wasm_decrypt_strings")
	if decrypt == nil {
		r.cleanupLocked()
		return fmt.Errorf("tsdk: missing mergewasm decrypt export")
	}
	for _, seg := range MergedDataSegments {
		ptr, length := seg[0], seg[1]
		if err := r.host.ensureBounds(mod.Memory(), ptr, length); err != nil {
			r.cleanupLocked()
			return err
		}
		if _, err := decrypt.Call(ctx, uint64(ptr), uint64(length), uint64(MergedDataKey)); err != nil {
			r.cleanupLocked()
			return fmt.Errorf("tsdk: decrypt segment %d+%d: %w", ptr, length, err)
		}
	}

	if _, err := mod.ExportedFunction("x").Call(ctx); err != nil {
		r.cleanupLocked()
		return fmt.Errorf("tsdk: export x: %w", err)
	}

	appKey, err := r.allocCStringLocked(TSDKAppKey)
	if err != nil {
		r.cleanupLocked()
		return err
	}
	_, gErr := mod.ExportedFunction("G").Call(ctx, uint64(TSDKGameID), uint64(appKey))
	r.freeLocked(appKey)
	if gErr != nil {
		r.cleanupLocked()
		return fmt.Errorf("tsdk: export G: %w", gErr)
	}

	r.ready = true
	return nil
}

func (r *Runtime) instantiateHost(ctx context.Context) error {
	h := r.host
	builder := r.runtime.NewHostModuleBuilder("a")
	builder.NewFunctionBuilder().WithFunc(h.aAssert).Export("a")
	builder.NewFunctionBuilder().WithFunc(h.bWriteFile).Export("b")
	builder.NewFunctionBuilder().WithFunc(h.cStack).Export("c")
	builder.NewFunctionBuilder().WithFunc(h.dVersion).Export("d")
	builder.NewFunctionBuilder().WithFunc(h.eAceVM).Export("e")
	builder.NewFunctionBuilder().WithFunc(h.fSensors).Export("f")
	builder.NewFunctionBuilder().WithFunc(h.gReadFile).Export("g")
	builder.NewFunctionBuilder().WithFunc(h.hClock).Export("h")
	builder.NewFunctionBuilder().WithFunc(h.iDataDir).Export("i")
	builder.NewFunctionBuilder().WithFunc(h.jDevice).Export("j")
	builder.NewFunctionBuilder().WithFunc(h.kRuntimeTable).Export("k")
	builder.NewFunctionBuilder().WithFunc(h.lPlatform).Export("l")
	builder.NewFunctionBuilder().WithFunc(h.mAppID).Export("m")
	builder.NewFunctionBuilder().WithFunc(h.nAppID2).Export("n")
	builder.NewFunctionBuilder().WithFunc(h.oIntegrity).Export("o")
	builder.NewFunctionBuilder().WithFunc(h.pStat).Export("p")
	builder.NewFunctionBuilder().WithFunc(h.qServerTime).Export("q")
	builder.NewFunctionBuilder().WithFunc(h.rMemFail).Export("r")
	builder.NewFunctionBuilder().WithFunc(h.sNow).Export("s")
	builder.NewFunctionBuilder().WithFunc(h.tAppendFile).Export("t")
	builder.NewFunctionBuilder().WithFunc(h.uAbort).Export("u")
	builder.NewFunctionBuilder().WithFunc(h.vTQOS).Export("v")
	_, err := builder.Instantiate(ctx)
	return err
}

func (r *Runtime) assertReadyLocked() error {
	if !r.ready || r.mod == nil || r.destroyed {
		return fmt.Errorf("tsdk: not ready")
	}
	return nil
}

func (r *Runtime) allocLocked(length uint32) (uint32, error) {
	if length == 0 {
		length = 1
	}
	results, err := r.mod.ExportedFunction("A").Call(r.ctx, uint64(length))
	if err != nil {
		return 0, err
	}
	ptr := uint32(results[0])
	if ptr == 0 {
		return 0, fmt.Errorf("tsdk: alloc failed size=%d", length)
	}
	if err := r.host.ensureBounds(r.mod.Memory(), ptr, length); err != nil {
		return 0, err
	}
	return ptr, nil
}

func (r *Runtime) allocBytesLocked(data []byte) (uint32, uint32, error) {
	n := uint32(len(data))
	if n == 0 {
		ptr, err := r.allocLocked(1)
		return ptr, 0, err
	}
	ptr, err := r.allocLocked(n)
	if err != nil {
		return 0, 0, err
	}
	if !r.mod.Memory().Write(ptr, data) {
		r.freeLocked(ptr)
		return 0, 0, fmt.Errorf("tsdk: write alloc failed")
	}
	return ptr, n, nil
}

func (r *Runtime) allocCStringLocked(s string) (uint32, error) {
	data := append([]byte(s), 0)
	ptr, _, err := r.allocBytesLocked(data)
	return ptr, err
}

func (r *Runtime) freeLocked(ptr uint32) {
	if ptr == 0 || r.mod == nil {
		return
	}
	_, _ = r.mod.ExportedFunction("B").Call(r.ctx, uint64(ptr))
}

// Encrypt encrypts buf in place via export ba. Returns a copy of the transformed bytes.
// TODO: verify ba availability / ABI against production captures.
func (r *Runtime) Encrypt(buf []byte) ([]byte, error) {
	return r.transform(buf, false)
}

// Decrypt decrypts buf in place via export ca. Returns a copy of the transformed bytes.
// TODO: verify ca availability / ABI against production captures.
func (r *Runtime) Decrypt(buf []byte) ([]byte, error) {
	return r.transform(buf, true)
}

func (r *Runtime) transform(buf []byte, decrypt bool) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.assertReadyLocked(); err != nil {
		return nil, err
	}
	ptr, length, err := r.allocBytesLocked(buf)
	if err != nil {
		return nil, err
	}
	defer r.freeLocked(ptr)

	name := "ba"
	if decrypt {
		name = "ca"
	}
	if _, err := r.mod.ExportedFunction(name).Call(r.ctx, uint64(ptr), uint64(length)); err != nil {
		return nil, fmt.Errorf("tsdk: %s: %w", name, err)
	}
	if err := r.host.ensureBounds(r.mod.Memory(), ptr, length); err != nil {
		return nil, err
	}
	out, ok := r.mod.Memory().Read(ptr, length)
	if !ok {
		return nil, fmt.Errorf("tsdk: read transformed bytes failed")
	}
	return append([]byte(nil), out...), nil
}

// BindUser re-calls G with openId once (matches Node bindUser).
func (r *Runtime) BindUser(openID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.assertReadyLocked(); err != nil {
		return err
	}
	openID = trimSpace(openID)
	if openID == "" || r.userBound {
		return nil
	}
	ptr, err := r.allocCStringLocked(openID)
	if err != nil {
		return err
	}
	defer r.freeLocked(ptr)
	if _, err := r.mod.ExportedFunction("G").Call(r.ctx, uint64(TSDKGameID), uint64(ptr)); err != nil {
		return err
	}
	r.userBound = true
	return nil
}

// GetEncryptedInitInfo returns H() C-string (one-shot TSDK credential; see docs).
func (r *Runtime) GetEncryptedInitInfo() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.assertReadyLocked(); err != nil {
		return "", err
	}
	results, err := r.mod.ExportedFunction("H").Call(r.ctx)
	if err != nil {
		return "", err
	}
	ptr := uint32(results[0])
	if ptr == 0 {
		return "", nil
	}
	return r.host.readCString(r.mod.Memory(), ptr)
}

// HeartbeatTick calls export M.
func (r *Runtime) HeartbeatTick() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.assertReadyLocked(); err != nil {
		return err
	}
	_, err := r.mod.ExportedFunction("M").Call(r.ctx)
	return err
}

// ProcessReceivedData calls export P.
func (r *Runtime) ProcessReceivedData() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.assertReadyLocked(); err != nil {
		return err
	}
	_, err := r.mod.ExportedFunction("P").Call(r.ctx)
	return err
}

// GetDataToServer returns ACE payload via export N (matches Node getDataToServer).
func (r *Runtime) GetDataToServer() ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.assertReadyLocked(); err != nil {
		return nil, err
	}
	lengthPtr, err := r.allocLocked(4)
	if err != nil {
		return nil, err
	}
	defer r.freeLocked(lengthPtr)
	mem := r.mod.Memory()
	if !mem.WriteUint32Le(lengthPtr, 0) {
		return nil, fmt.Errorf("tsdk: write length ptr failed")
	}
	results, err := r.mod.ExportedFunction("N").Call(r.ctx, uint64(lengthPtr))
	if err != nil {
		return nil, fmt.Errorf("tsdk: N: %w", err)
	}
	dataPtr := uint32(results[0])
	length, ok := mem.ReadUint32Le(lengthPtr)
	if !ok || dataPtr == 0 || length == 0 {
		return nil, nil
	}
	if err := r.host.ensureBounds(mem, dataPtr, length); err != nil {
		return nil, err
	}
	out, ok := mem.Read(dataPtr, length)
	if !ok {
		return nil, fmt.Errorf("tsdk: read N payload failed")
	}
	return append([]byte(nil), out...), nil
}

// SendDataFromServer feeds ACE reply via export O.
func (r *Runtime) SendDataFromServer(data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.assertReadyLocked(); err != nil {
		return err
	}
	ptr, length, err := r.allocBytesLocked(data)
	if err != nil {
		return err
	}
	defer r.freeLocked(ptr)
	_, err = r.mod.ExportedFunction("O").Call(r.ctx, uint64(ptr), uint64(length))
	return err
}

// SendStatus calls export E.
func (r *Runtime) SendStatus() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.assertReadyLocked(); err != nil {
		return err
	}
	_, err := r.mod.ExportedFunction("E").Call(r.ctx)
	return err
}

// DetectSpeedHack calls export fa with elapsed ms.
func (r *Runtime) DetectSpeedHack(elapsedMs int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.assertReadyLocked(); err != nil {
		return err
	}
	if elapsedMs < 0 {
		elapsedMs = 0
	}
	_, err := r.mod.ExportedFunction("fa").Call(r.ctx, uint64(elapsedMs))
	return err
}

// Destroy tears down the WASM instance.
func (r *Runtime) Destroy() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ready = false
	r.destroyed = true
	r.host.mu.Lock()
	r.host.serverTimeGeneration++
	r.host.mu.Unlock()
	r.cleanupLocked()
}

func (r *Runtime) cleanupLocked() {
	if r.mod != nil {
		_ = r.mod.Close(r.ctx)
		r.mod = nil
	}
	if r.runtime != nil {
		_ = r.runtime.Close(r.ctx)
		r.runtime = nil
	}
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.ready = false
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 {
		c := s[len(s)-1]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}
