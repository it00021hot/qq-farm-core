package hub

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

const logRingCap = 1000

// LogMeta carries module/event for dashboard filters.
type LogMeta struct {
	Module string `json:"module,omitempty"`
	Event  string `json:"event,omitempty"`
}

// LogEntry is one run-log line (aligned with qq-farm-bot LogEntry).
type LogEntry struct {
	Time      string  `json:"time"`
	Tag       string  `json:"tag"`
	Msg       string  `json:"msg"`
	IsWarn    bool    `json:"isWarn"`
	Meta      LogMeta `json:"meta"`
	AccountID uint64  `json:"accountId"`
	Ts        int64   `json:"ts"`
}

// LogStore is a per-account in-memory ring buffer (cap 1000 each).
type LogStore struct {
	mu     sync.RWMutex
	byAcct map[uint64][]LogEntry
}

// Logs is the process-wide run-log store.
var Logs = NewLogStore()

func NewLogStore() *LogStore {
	return &LogStore{byAcct: make(map[uint64][]LogEntry)}
}

// Append pushes an entry into the account ring (drops oldest past cap).
func (s *LogStore) Append(e LogEntry) {
	if e.AccountID == 0 {
		return
	}
	if e.Ts == 0 {
		e.Ts = time.Now().UnixMilli()
	}
	if e.Time == "" {
		e.Time = time.UnixMilli(e.Ts).Local().Format("2006-01-02 15:04:05")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	buf := s.byAcct[e.AccountID]
	buf = append(buf, e)
	if len(buf) > logRingCap {
		buf = append([]LogEntry(nil), buf[len(buf)-logRingCap:]...)
	}
	s.byAcct[e.AccountID] = buf
}

// Query returns the last limit entries matching filters (oldest→newest).
// accountID 0 merges all accounts (still sorted by ts ascending within the slice window).
func (s *LogStore) Query(accountID uint64, module, keyword string, limit int) []LogEntry {
	if limit <= 0 {
		limit = 100
	}
	module = strings.TrimSpace(module)
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	terms := strings.Fields(keyword)

	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []LogEntry
	if accountID != 0 {
		list = append([]LogEntry(nil), s.byAcct[accountID]...)
	} else {
		for _, buf := range s.byAcct {
			list = append(list, buf...)
		}
		if len(list) > 1 {
			// Stable enough: sort by ts ascending for merged view.
			sortLogEntries(list)
		}
	}

	out := make([]LogEntry, 0, len(list))
	for _, e := range list {
		if module != "" {
			logMod := e.Meta.Module
			if module == "system" {
				if logMod != "system" && e.Tag != "系统" && e.Tag != "错误" {
					continue
				}
			} else if logMod != module {
				continue
			}
		}
		if len(terms) > 0 {
			text := strings.ToLower(e.Msg + " " + e.Tag + " " + e.Meta.Event + " " + e.Meta.Module)
			ok := true
			for _, t := range terms {
				if !strings.Contains(text, t) {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
		}
		out = append(out, e)
	}
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

// Clear removes logs for one account (or all when accountID is 0).
func (s *LogStore) Clear(accountID uint64) (cleared int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if accountID == 0 {
		for id, buf := range s.byAcct {
			cleared += len(buf)
			delete(s.byAcct, id)
		}
		return cleared
	}
	cleared = len(s.byAcct[accountID])
	delete(s.byAcct, accountID)
	return cleared
}

func sortLogEntries(list []LogEntry) {
	// insertion sort — rings are small (≤1000×N) and usually nearly sorted
	for i := 1; i < len(list); i++ {
		j := i
		for j > 0 && list[j-1].Ts > list[j].Ts {
			list[j-1], list[j] = list[j], list[j-1]
			j--
		}
	}
}

// AppendFromEvent converts a hub publish into a log entry when the type is log-worthy.
func AppendFromEvent(typ string, accountID uint64, payload any) {
	entry, ok := logEntryFromEvent(typ, accountID, payload)
	if !ok {
		return
	}
	Logs.Append(entry)
}

func logEntryFromEvent(typ string, accountID uint64, payload any) (LogEntry, bool) {
	switch typ {
	case "farm_operation", "farm_tick", "friend_interact", "account_status":
	default:
		return LogEntry{}, false
	}
	m := payloadToMap(payload)
	now := time.Now()
	entry := LogEntry{
		Time:      now.Format("2006-01-02 15:04:05"),
		Ts:        now.UnixMilli(),
		AccountID: accountID,
		IsWarn:    asBool(m["isWarn"]),
		Tag:       asString(m["tag"]),
		Msg:       asString(m["message"]),
		Meta: LogMeta{
			Event: asString(m["event"]),
		},
	}
	switch typ {
	case "farm_operation", "farm_tick":
		entry.Meta.Module = "farm"
		if entry.Tag == "" {
			if entry.IsWarn {
				entry.Tag = "错误"
			} else {
				entry.Tag = "农场"
			}
		}
		if entry.Meta.Event == "" {
			if typ == "farm_tick" {
				entry.Meta.Event = "农场巡查"
			} else {
				entry.Meta.Event = "农场操作"
			}
		}
		if entry.Msg == "" {
			entry.Msg = joinActions(m)
		}
	case "friend_interact":
		entry.Meta.Module = "friend"
		if entry.Tag == "" {
			entry.Tag = "好友"
		}
		if entry.Msg == "" {
			entry.Msg = asString(m["action"])
		}
	case "account_status":
		entry.Meta.Module = "system"
		st := asString(m["status"])
		detail := asString(m["detail"])
		entry.IsWarn = st == "error"
		if entry.Tag == "" {
			if entry.IsWarn {
				entry.Tag = "错误"
			} else {
				entry.Tag = "系统"
			}
		}
		entry.Meta.Event = "账号状态"
		if entry.Msg == "" {
			entry.Msg = formatAccountStatusMsg(st, detail)
		}
	}
	return entry, true
}

func payloadToMap(payload any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	if m, ok := payload.(map[string]any); ok {
		return m
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil || m == nil {
		return map[string]any{}
	}
	return m
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

func asBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "1" || strings.EqualFold(t, "true")
	default:
		return false
	}
}

func joinActions(m map[string]any) string {
	raw, ok := m["actions"]
	if !ok {
		if errText := asString(m["error"]); errText != "" {
			return errText
		}
		op := asString(m["op"])
		switch op {
		case "", "all":
			return "巡查完成"
		default:
			return op
		}
	}
	switch a := raw.(type) {
	case []string:
		return strings.Join(a, "/")
	case []any:
		parts := make([]string, 0, len(a))
		for _, x := range a {
			s := asString(x)
			if s != "" {
				parts = append(parts, s)
			}
		}
		if len(parts) == 0 {
			return "巡查完成"
		}
		return strings.Join(parts, "/")
	default:
		s := asString(raw)
		if s == "" || s == "all" {
			return "巡查完成"
		}
		return s
	}
}

func formatAccountStatusMsg(st, detail string) string {
	label := st
	switch st {
	case "running":
		label = "运行中"
	case "starting":
		label = "启动中"
	case "stopped":
		label = "已停止"
	case "stopping":
		label = "停止中"
	case "error":
		label = "异常"
	}
	if detail == "" {
		if label == "" {
			return "-"
		}
		return label
	}
	if label == "" {
		return detail
	}
	return label + " · " + detail
}
