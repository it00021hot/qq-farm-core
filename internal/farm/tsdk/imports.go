package tsdk

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/tetratelabs/wazero/api"
)

// host implements WASM imports a.a–a.v with Node tsdk-runtime.ts degradation semantics.
type host struct {
	rt *Runtime

	mu                   sync.Mutex
	warned               map[string]struct{}
	serverTimeGeneration uint32
	monoStart            time.Time
}

func newHost(rt *Runtime) *host {
	return &host{rt: rt, warned: make(map[string]struct{}), monoStart: time.Now()}
}

func (h *host) warnOnce(key, msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.warned[key]; ok {
		return
	}
	h.warned[key] = struct{}{}
	slog.Warn(msg, "component", "tsdk", "key", key)
}

func memOf(m api.Module) api.Memory {
	if m == nil {
		return nil
	}
	return m.Memory()
}

func (h *host) ensureBounds(m api.Memory, ptr, length uint32) error {
	if m == nil {
		return fmt.Errorf("TSDK memory not ready")
	}
	size := m.Size()
	if int64(ptr)+int64(length) > int64(size) {
		return fmt.Errorf("TSDK OOB: ptr=%d length=%d size=%d", ptr, length, size)
	}
	return nil
}

func (h *host) readCString(m api.Memory, ptr uint32) (string, error) {
	if ptr == 0 {
		return "", nil
	}
	if m == nil {
		return "", fmt.Errorf("TSDK memory not ready")
	}
	size := m.Size()
	if ptr >= size {
		return "", fmt.Errorf("TSDK string ptr OOB")
	}
	limit := size
	if limit > ptr+1024*1024 {
		limit = ptr + 1024*1024
	}
	buf, ok := m.Read(ptr, limit-ptr)
	if !ok {
		return "", fmt.Errorf("TSDK string read failed")
	}
	end := 0
	for end < len(buf) && buf[end] != 0 {
		end++
	}
	if end >= len(buf) {
		return "", fmt.Errorf("TSDK string not NUL-terminated")
	}
	return string(buf[:end]), nil
}

func (h *host) writeCString(m api.Memory, value string, ptr, capacity uint32) uint32 {
	data := []byte(value)
	if ptr == 0 || capacity <= uint32(len(data)) {
		return 0
	}
	if err := h.ensureBounds(m, ptr, capacity); err != nil {
		return 0
	}
	out := make([]byte, len(data)+1)
	copy(out, data)
	if !m.Write(ptr, out) {
		return 0
	}
	return ptr
}

func (h *host) writeBytes(m api.Memory, value []byte, ptr, capacity uint32) uint32 {
	if ptr == 0 || capacity < uint32(len(value)) {
		return 0
	}
	if err := h.ensureBounds(m, ptr, capacity); err != nil {
		return 0
	}
	if !m.Write(ptr, value) {
		return 0
	}
	return uint32(len(value))
}

func (h *host) resolveDataPath(input string) (string, error) {
	rel := strings.ReplaceAll(input, "\\", "/")
	rel = strings.TrimLeft(rel, "/")
	root, err := filepath.Abs(h.rt.dataDir)
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(filepath.Join(root, rel))
	if err != nil {
		return "", err
	}
	if target != root && !strings.HasPrefix(target, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("TSDK path escapes account dir")
	}
	return target, nil
}

func (h *host) deviceText() string {
	model := h.rt.deviceModel
	if model == "" {
		model = runtime.GOOS + " " + runtime.GOARCH
	}
	platform := h.rt.osName
	if platform == "" {
		platform = runtime.GOOS
	}
	system := h.rt.sysSoftware
	if system == "" {
		system = "unknown"
	}
	return fmt.Sprintf("%s;%s;%s;Go;", model, platform, system)
}

// --- host imports a.a – a.v (first arg after ctx is calling module) ---

func (h *host) aAssert(_ context.Context, mod api.Module, exprPtr, filePtr, line, funcPtr uint32) {
	mem := memOf(mod)
	expr, _ := h.readCString(mem, exprPtr)
	file, _ := h.readCString(mem, filePtr)
	if file == "" {
		file = "unknown"
	}
	fn, _ := h.readCString(mem, funcPtr)
	panic(fmt.Sprintf("TSDK assertion: %s at %s:%d %s", expr, file, line, fn))
}

func (h *host) bWriteFile(_ context.Context, mod api.Module, filePtr, dataPtr, encodingPtr uint32) uint32 {
	mem := memOf(mod)
	pathStr, err := h.readCString(mem, filePtr)
	if err != nil {
		h.warnOnce("write-file", "TSDK file write failed: "+err.Error())
		return 0
	}
	target, err := h.resolveDataPath(pathStr)
	if err != nil {
		h.warnOnce("write-file", "TSDK file write failed: "+err.Error())
		return 0
	}
	data, err := h.readCString(mem, dataPtr)
	if err != nil {
		return 0
	}
	_ = encodingPtr
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0
	}
	if err := os.WriteFile(target, []byte(data), 0o644); err != nil {
		h.warnOnce("write-file", "TSDK file write failed: "+err.Error())
		return 0
	}
	return 1
}

func (h *host) cStack(_ context.Context, mod api.Module, ptr, capacity uint32) uint32 {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	stack := string(buf[:n])
	// Truncate so write always succeeds when capacity > 1 (Node stacks are short;
	// a failed write previously caused G() to OOB).
	maxLen := int(capacity) - 1
	if maxLen < 0 {
		maxLen = 0
	}
	if len(stack) > maxLen {
		stack = stack[:maxLen]
	}
	if h.writeCString(memOf(mod), stack, ptr, capacity) == 0 {
		return 0
	}
	return uint32(len(stack) + 1)
}

func (h *host) dVersion(_ context.Context, mod api.Module, ptr, capacity uint32) uint32 {
	return h.writeCString(memOf(mod), TSDKVersion, ptr, capacity)
}

func (h *host) eAceVM(_ context.Context, mod api.Module, outPtr uint32) uint32 {
	h.warnOnce("acevm", "Go host does not provide mini-game ACEVM integrity context; using empty result")
	// Official hosts return an empty result buffer; leave zeros at outPtr when provided.
	if outPtr != 0 {
		if mem := memOf(mod); mem != nil {
			// Best-effort: clear a small header so callers see an empty payload.
			var zeros [16]byte
			if err := h.ensureBounds(mem, outPtr, uint32(len(zeros))); err == nil {
				mem.Write(outPtr, zeros[:])
			}
		}
	}
	return 0
}

func (h *host) fSensors(_ context.Context, _ api.Module) {
	h.warnOnce("sensors", "Go host does not provide touch/gyroscope data")
}

func (h *host) gReadFile(_ context.Context, mod api.Module, filePtr, outputPtr, capacity, encodingPtr uint32) uint32 {
	mem := memOf(mod)
	pathStr, err := h.readCString(mem, filePtr)
	if err != nil {
		return 0
	}
	target, err := h.resolveDataPath(pathStr)
	if err != nil {
		return 0
	}
	_ = encodingPtr
	raw, err := os.ReadFile(target)
	if err != nil {
		return 0
	}
	return h.writeCString(mem, string(raw), outputPtr, capacity)
}

func (h *host) hClock(_ context.Context, mod api.Module, clockID, _, _, outputPtr uint32) uint32 {
	if clockID > 3 {
		return 28
	}
	mem := memOf(mod)
	var value int64
	if clockID == 0 {
		// wall clock: Date.now() * 1e6 (ns-ish units matching Node)
		value = time.Now().UnixMilli() * 1_000_000
	} else {
		// monotonic: performance.now() * 1e6
		value = time.Since(h.monoStart).Nanoseconds() / 1000 * 1000 // µs * 1e6? Node: performance.now()*1e6
		value = int64(float64(time.Since(h.monoStart).Milliseconds()) * 1e6)
	}
	if err := h.ensureBounds(mem, outputPtr, 8); err != nil {
		return 28
	}
	var buf [8]byte
	binary.LittleEndian.PutUint32(buf[0:4], uint32(value))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(uint64(value)>>32))
	if !mem.Write(outputPtr, buf[:]) {
		return 28
	}
	return 0
}

func (h *host) iDataDir(_ context.Context, mod api.Module, ptr, capacity uint32) uint32 {
	dir := h.rt.dataDir
	if !strings.HasSuffix(dir, string(os.PathSeparator)) {
		dir += string(os.PathSeparator)
	}
	return h.writeCString(memOf(mod), dir, ptr, capacity)
}

func (h *host) jDevice(_ context.Context, mod api.Module, ptr, capacity uint32) uint32 {
	return h.writeCString(memOf(mod), h.deviceText(), ptr, capacity)
}

func (h *host) kRuntimeTable(_ context.Context, mod api.Module, ptr, capacity uint32) uint32 {
	return h.writeBytes(memOf(mod), RuntimeTable, ptr, capacity)
}

func (h *host) lPlatform(_ context.Context, _ api.Module) uint32 { return 2 }

func (h *host) mAppID(_ context.Context, mod api.Module, ptr, capacity uint32) uint32 {
	return h.writeCString(memOf(mod), MiniProgramAppID, ptr, capacity)
}

func (h *host) nAppID2(_ context.Context, mod api.Module, ptr, capacity uint32) uint32 {
	return h.writeCString(memOf(mod), MiniProgramAppID, ptr, capacity)
}

func (h *host) oIntegrity(_ context.Context, _ api.Module, _, _, _, _ uint32) {
	// Node leaves the caller-provided buffer untouched and only logs once.
	h.warnOnce("integrity-functions", "Go host does not provide mini-game function integrity list")
}

func (h *host) pStat(_ context.Context, mod api.Module, filePtr uint32) uint32 {
	mem := memOf(mod)
	pathStr, err := h.readCString(mem, filePtr)
	if err != nil {
		return 0
	}
	target, err := h.resolveDataPath(pathStr)
	if err != nil {
		return 0
	}
	st, err := os.Stat(target)
	if err != nil {
		return 0
	}
	mode := uint32(st.Mode().Perm())
	size := st.Size()
	if size > math.MaxInt32 {
		size = math.MaxInt32
	}
	atime := st.ModTime().UnixMilli()
	mtime := st.ModTime().UnixMilli()
	fn := mod.ExportedFunction("y")
	if fn == nil {
		return 0
	}
	results, err := fn.Call(context.Background(), uint64(mode), uint64(size), uint64(atime), uint64(mtime))
	if err != nil || len(results) == 0 {
		return 0
	}
	return uint32(results[0])
}

func (h *host) qServerTime(_ context.Context, mod api.Module, outputPtr uint32) uint32 {
	h.mu.Lock()
	h.serverTimeGeneration++
	gen := h.serverTimeGeneration
	h.mu.Unlock()

	mem := memOf(mod)
	if err := h.ensureBounds(mem, outputPtr, 4); err != nil {
		return 0
	}
	now := uint32(time.Now().Unix())
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], now)
	mem.Write(outputPtr, buf[:])

	go func() {
		client := &http.Client{
			Timeout: 3 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			},
		}
		resp, err := client.Get("https://api.anticheatexpert.com/test")
		if err != nil {
			return
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body) //nolint:errcheck

		h.mu.Lock()
		ok := gen == h.serverTimeGeneration
		h.mu.Unlock()
		if !ok {
			return
		}
		parsed, err := http.ParseTime(resp.Header.Get("Date"))
		var sec uint32
		if err == nil {
			sec = uint32(parsed.Unix())
		}
		var out [4]byte
		binary.LittleEndian.PutUint32(out[:], sec)
		// Best-effort write back via runtime module if still alive.
		if h.rt.mod != nil {
			if m := h.rt.mod.Memory(); m != nil {
				m.Write(outputPtr, out[:])
			}
		}
	}()
	return 1
}

func (h *host) rMemFail(_ context.Context, _ api.Module, size uint32) uint32 {
	panic(fmt.Sprintf("TSDK memory grow failed: %d", size))
}

func (h *host) sNow(_ context.Context, _ api.Module) float64 {
	return float64(time.Now().UnixMilli())
}

func (h *host) tAppendFile(_ context.Context, mod api.Module, filePtr, dataPtr, encodingPtr uint32) uint32 {
	mem := memOf(mod)
	pathStr, err := h.readCString(mem, filePtr)
	if err != nil {
		return 0
	}
	target, err := h.resolveDataPath(pathStr)
	if err != nil {
		return 0
	}
	data, err := h.readCString(mem, dataPtr)
	if err != nil {
		return 0
	}
	_ = encodingPtr
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0
	}
	f, err := os.OpenFile(target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0
	}
	defer f.Close()
	if _, err := f.WriteString(data); err != nil {
		return 0
	}
	return 1
}

func (h *host) uAbort(_ context.Context, _ api.Module) {
	panic("TSDK aborted")
}

func (h *host) vTQOS(_ context.Context, mod api.Module, ptr, length uint32) uint32 {
	mem := memOf(mod)
	if err := h.ensureBounds(mem, ptr, length); err != nil {
		h.warnOnce("tqos", "TSDK TQOS data invalid: "+err.Error())
		return 0
	}
	raw, ok := mem.Read(ptr, length)
	if !ok {
		return 0
	}
	var report struct {
		Headers map[string]string `json:"headers"`
		Message json.RawMessage   `json:"message"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		h.warnOnce("tqos", "TSDK TQOS data invalid: "+err.Error())
		return 0
	}
	go func() {
		req, err := http.NewRequest(http.MethodPost, "https://api.anticheatexpert.com/tqos", strings.NewReader(string(report.Message)))
		if err != nil {
			return
		}
		for k, v := range report.Headers {
			req.Header.Set(k, v)
		}
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			h.warnOnce("tqos", "TSDK TQOS report failed: "+err.Error())
			return
		}
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
		resp.Body.Close()
	}()
	return 0
}
