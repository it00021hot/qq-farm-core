package runtime

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/farm/game"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/userpb"
)

const inviteRequestDelay = 2 * time.Second
const inviteSceneID = "1256"

// parsedShareLink mirrors Node invite.ts ParsedShareLink.
type parsedShareLink struct {
	uid        string
	openid     string
	shareCfgID int64
}

// resolveShareFilePath mirrors Node getShareFilePath(): env FARM_SHARE_FILE,
// then configured farm.shareFile, else share.txt under DataRoot.
func resolveShareFilePath(dataRoot, configuredPath string) string {
	if p := strings.TrimSpace(os.Getenv("FARM_SHARE_FILE")); p != "" {
		return p
	}
	if p := strings.TrimSpace(configuredPath); p != "" {
		return p
	}
	root := strings.TrimSpace(dataRoot)
	if root == "" {
		root = "data"
	}
	return filepath.Join(root, "share.txt")
}

// parseShareLink extracts uid / openid / share_source from a query-like line.
// Formats: ?uid=…&openid=…&share_source=… or a full URL with the same query.
func parseShareLink(link string) parsedShareLink {
	link = strings.TrimSpace(link)
	if link == "" {
		return parsedShareLink{}
	}
	query := link
	if i := strings.Index(link, "?"); i >= 0 {
		query = link[i+1:]
	}
	vals, err := url.ParseQuery(query)
	if err != nil {
		return parsedShareLink{}
	}
	out := parsedShareLink{
		uid:    vals.Get("uid"),
		openid: vals.Get("openid"),
	}
	if src := strings.TrimSpace(vals.Get("share_source")); src != "" {
		if n, err := strconv.ParseInt(src, 10, 64); err == nil {
			out.shareCfgID = n
		}
	}
	return out
}

// readShareFile reads and dedupes invite lines (must contain openid=), matching Node.
func readShareFile(path string) []parsedShareLink {
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Debug("invite: read share.txt failed", "path", path, "err", err)
		}
		return nil
	}
	seen := make(map[string]struct{})
	var out []parsedShareLink
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "openid=") {
			continue
		}
		p := parseShareLink(line)
		if p.openid == "" || p.uid == "" {
			continue
		}
		if _, ok := seen[p.uid]; ok {
			continue
		}
		seen[p.uid] = struct{}{}
		out = append(out, p)
	}
	return out
}

func clearShareFile(path string) {
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		slog.Debug("invite: clear share.txt failed", "path", path, "err", err)
		return
	}
	slog.Info("invite: cleared share.txt", "path", path)
}

// processInviteCodes mirrors Node invite.ts processInviteCodes (WX-only ReportArkClick).
func processInviteCodes(ctx context.Context, api *game.API, dataRoot, shareFile string) {
	if api == nil {
		return
	}
	path := resolveShareFilePath(dataRoot, shareFile)
	invites := readShareFile(path)
	if len(invites) == 0 {
		return
	}
	slog.Info("invite: processing share links", "count", len(invites), "path", path)

	ok, fail := 0, 0
	for i, inv := range invites {
		sharerID, err := strconv.ParseInt(inv.uid, 10, 64)
		if err != nil {
			fail++
			slog.Warn("invite: bad uid", "uid", inv.uid, "err", err)
			continue
		}
		_, err = api.ReportArkClick(ctx, &userpb.ReportArkClickRequest{
			SharerID:     sharerID,
			SharerOpenID: inv.openid,
			ShareCfgID:   inv.shareCfgID,
			SceneID:      inviteSceneID,
		})
		if err != nil {
			fail++
			slog.Warn("invite: ReportArkClick failed",
				"i", i+1, "n", len(invites), "uid", inv.uid, "err", err)
		} else {
			ok++
			slog.Info("invite: ReportArkClick sent",
				"i", i+1, "n", len(invites), "uid", inv.uid)
		}
		if i < len(invites)-1 {
			select {
			case <-ctx.Done():
				slog.Warn("invite: aborted", "ok", ok, "fail", fail)
				return
			case <-time.After(inviteRequestDelay):
			}
		}
	}
	slog.Info("invite: done", "ok", ok, "fail", fail)
	clearShareFile(path)
}
