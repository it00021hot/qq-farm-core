package appserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v3"
)

func TestMountWebUIServesSPAWithoutEatingAPI(t *testing.T) {
	webFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>spa</html>")},
		"assets/a.js": &fstest.MapFile{Data: []byte("console.log(1)")},
	}
	app := fiber.New()
	app.Get("/ping", func(c fiber.Ctx) error { return c.SendString("pong") })
	app.Get("/farm/ws", func(c fiber.Ctx) error { return c.SendString("ws") })
	mountWebUI(app, webFS)

	assertBody := func(path, want string, status int) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != status {
			t.Fatalf("%s status=%d want %d body=%q", path, resp.StatusCode, status, body)
		}
		if !strings.Contains(string(body), want) {
			t.Fatalf("%s body=%q want contain %q", path, body, want)
		}
	}

	assertBody("/ping", "pong", 200)
	assertBody("/farm/ws", "ws", 200)
	assertBody("/", "spa", 200)
	assertBody("/assets/a.js", "console.log", 200)
	assertBody("/some/client/route", "spa", 200)
}
