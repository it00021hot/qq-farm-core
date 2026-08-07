package appserver

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/bootstrap"
	farmruntime "github.com/it00021hot/qq-farm-core/internal/farm/runtime"
	"github.com/it00021hot/qq-farm-core/internal/router"
	"github.com/it00021hot/qq-farm-core/internal/vars"
	"github.com/it00021hot/qq-farm-core/pkg/config"
	"github.com/gofiber/fiber/v3"
)

const (
	DefaultHost = "127.0.0.1"
	DefaultPort = "9528"
	AppDataName = "QQFarm"
)

// Options configures the in-process API for desktop (or embedded) hosts.
type Options struct {
	Env          string // config env: dev|test|prod
	Host         string
	Port         string
	ResourceRoot string
	DataRoot     string
	DesktopMode  bool
}

// Server is a running Fiber API bound to loopback.
type Server struct {
	app  *fiber.App
	addr string
}

func (s *Server) Addr() string {
	if s == nil {
		return ""
	}
	return s.addr
}

func (s *Server) BaseURL() string {
	return "http://" + s.Addr()
}

// Start boots config/DB/farm runtime and listens on host:port (default 127.0.0.1:9528).
func Start(opts Options) (*Server, error) {
	if opts.DesktopMode {
		vars.DesktopMode = true
	}
	resourceRoot := opts.ResourceRoot
	if resourceRoot == "" {
		var err error
		resourceRoot, err = ResolveResourceRoot()
		if err != nil {
			return nil, err
		}
	}
	dataRoot := opts.DataRoot
	if dataRoot == "" {
		var err error
		dataRoot, err = ResolveDataRoot()
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	vars.SetPaths(resourceRoot, dataRoot)
	log.Printf("appserver paths: resource=%s data=%s", resourceRoot, dataRoot)

	if opts.Env == "" {
		opts.Env = "prod"
	}
	config.ConfEnv = opts.Env
	bootstrap.BootService()

	host := opts.Host
	if host == "" {
		host = DefaultHost
	}
	port := opts.Port
	if port == "" {
		port = DefaultPort
	}
	addr := host + ":" + port
	app := router.Register("qq-farm-desktop")

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Listen(addr, fiber.ListenConfig{
			EnablePrefork:         false,
			DisableStartupMessage: false,
		})
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("fiber listen %s: %w", addr, err)
		}
	case <-time.After(400 * time.Millisecond):
	}

	return &Server{app: app, addr: addr}, nil
}

// Shutdown stops farm sessions and the Fiber listener.
func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	farmruntime.Default.StopAll()
	if s.app == nil {
		return nil
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}
	return s.app.ShutdownWithContext(ctx)
}

// ResolveResourceRoot finds bundled farm resources (wasm/gameConfig).
func ResolveResourceRoot() (string, error) {
	if v := os.Getenv("QQFARM_RESOURCE_ROOT"); v != "" {
		return filepath.Clean(v), nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	exeDir := filepath.Dir(exe)

	resources := filepath.Clean(filepath.Join(exeDir, "..", "Resources"))
	if isFarmResourceDir(resources) {
		return resources, nil
	}
	if isFarmResourceDir(exeDir) {
		return exeDir, nil
	}

	candidates := []string{
		filepath.Join(exeDir, "..", "..", "qq-farm-core"),
		filepath.Join(exeDir, "..", "qq-farm-core"),
		filepath.Join(exeDir, "qq-farm-core"),
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "..", "qq-farm-core"),
			filepath.Join(wd, "qq-farm-core"),
		)
	}
	for _, c := range candidates {
		c = filepath.Clean(c)
		if isFarmResourceDir(c) {
			return c, nil
		}
	}
	return exeDir, nil
}

// ResolveDataRoot returns the writable application data directory.
func ResolveDataRoot() (string, error) {
	if v := os.Getenv("QQFARM_DATA_ROOT"); v != "" {
		return filepath.Clean(v), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch {
	case fileExists("/System/Library/CoreServices/SystemVersion.plist"):
		return filepath.Join(home, "Library", "Application Support", AppDataName), nil
	case os.Getenv("LOCALAPPDATA") != "":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), AppDataName), nil
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, AppDataName), nil
		}
		return filepath.Join(home, ".local", "share", AppDataName), nil
	}
}

func isFarmResourceDir(root string) bool {
	st, err := os.Stat(filepath.Join(root, "resource", "farm"))
	return err == nil && st.IsDir()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
