package hub

import (
	"testing"
)

func TestLogStoreAppendQueryClear(t *testing.T) {
	s := NewLogStore()
	for i := 0; i < 5; i++ {
		s.Append(LogEntry{
			AccountID: 1,
			Tag:       "农场",
			Msg:       "harvest",
			Meta:      LogMeta{Module: "farm", Event: "农场操作"},
			Ts:        int64(1000 + i),
		})
	}
	s.Append(LogEntry{
		AccountID: 1,
		Tag:       "好友",
		Msg:       "steal",
		Meta:      LogMeta{Module: "friend", Event: "偷好友菜"},
		Ts:        2000,
	})
	s.Append(LogEntry{
		AccountID: 2,
		Tag:       "系统",
		Msg:       "运行中",
		Meta:      LogMeta{Module: "system", Event: "账号状态"},
		Ts:        3000,
	})

	got := s.Query(1, "farm", "", 10)
	if len(got) != 5 {
		t.Fatalf("farm filter want 5 got %d", len(got))
	}
	got = s.Query(1, "", "steal", 10)
	if len(got) != 1 || got[0].Msg != "steal" {
		t.Fatalf("keyword filter: %+v", got)
	}
	got = s.Query(1, "", "", 3)
	if len(got) != 3 {
		t.Fatalf("limit want 3 got %d", len(got))
	}
	n := s.Clear(1)
	if n != 6 {
		t.Fatalf("cleared want 6 got %d", n)
	}
	if len(s.Query(1, "", "", 10)) != 0 {
		t.Fatal("account 1 should be empty")
	}
	if len(s.Query(2, "", "", 10)) != 1 {
		t.Fatal("account 2 should remain")
	}
}

func TestLogStoreRingCap(t *testing.T) {
	s := NewLogStore()
	for i := 0; i < logRingCap+50; i++ {
		s.Append(LogEntry{AccountID: 9, Msg: "x", Ts: int64(i + 1)})
	}
	got := s.Query(9, "", "", logRingCap+100)
	if len(got) != logRingCap {
		t.Fatalf("cap want %d got %d", logRingCap, len(got))
	}
	if got[0].Ts != 51 {
		t.Fatalf("oldest ts want 51 got %d", got[0].Ts)
	}
}

func TestAppendFromEvent(t *testing.T) {
	Logs = NewLogStore()
	AppendFromEvent("status", 1, map[string]any{"online": true})
	if len(Logs.Query(1, "", "", 10)) != 0 {
		t.Fatal("status should not append")
	}
	AppendFromEvent("farm_operation", 1, map[string]any{
		"tag": "农场", "event": "农场操作", "message": "harvest", "isWarn": false,
	})
	AppendFromEvent("account_status", 1, map[string]any{
		"status": "error", "detail": "被踢下线",
	})
	got := Logs.Query(1, "", "", 10)
	if len(got) != 2 {
		t.Fatalf("want 2 got %d", len(got))
	}
	if got[1].Meta.Module != "system" || !got[1].IsWarn {
		t.Fatalf("account_status entry: %+v", got[1])
	}
	if got[0].Meta.Module != "farm" || got[0].Msg != "harvest" {
		t.Fatalf("farm_operation entry: %+v", got[0])
	}
}
