package appserver

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

// mountWebUI serves a Vue SPA from webFS (dist root: index.html + assets/).
// API routes registered earlier take precedence; unknown GET paths fall back to index.html.
func mountWebUI(app *fiber.App, webFS fs.FS) {
	if app == nil || webFS == nil {
		return
	}

	indexHTML, err := fs.ReadFile(webFS, "index.html")
	if err != nil {
		return
	}

	app.Use(static.New("", static.Config{
		FS:     webFS,
		Browse: false,
		Next: func(c fiber.Ctx) bool {
			method := c.Method()
			if method != http.MethodGet && method != http.MethodHead {
				return true
			}
			return isReservedAPIPath(c.Path())
		},
		NotFoundHandler: func(c fiber.Ctx) error {
			if isReservedAPIPath(c.Path()) {
				return fiber.ErrNotFound
			}
			c.Status(fiber.StatusOK)
			c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
			return c.Send(indexHTML)
		},
	}))
}

// isReservedAPIPath reports paths that belong to the Fiber API (or docs), not the SPA.
func isReservedAPIPath(path string) bool {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return false
	}
	first, _, _ := strings.Cut(path, "/")
	switch first {
	case "auth", "farm", "system", "api", "game-config", "token", "ping", "docs", "swagger":
		return true
	default:
		return false
	}
}
