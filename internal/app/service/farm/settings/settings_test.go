package settings

import "testing"

func TestResolveClientVersionGuard(t *testing.T) {
	def := defaultClientVerUpdatedAtMs
	if def != 1_789_352_998_016 {
		t.Fatalf("default updatedAt = %d", def)
	}
	// 已保存版本更旧（updatedAt == 默认）→ 回默认版本。
	gotVer, gotAt := ResolveClientVersion("1.13.2.10_20260723", def)
	if gotVer != defaultClientVer || gotAt != def {
		t.Fatalf("stale saved: got (%q,%d)", gotVer, gotAt)
	}
	// 已保存版本同默认但无时间戳 → 回默认。
	gotVer, _ = ResolveClientVersion(defaultClientVer, 0)
	if gotVer != defaultClientVer {
		t.Fatalf("no timestamp: got %q", gotVer)
	}
	// 已保存版本更新（updatedAt > 默认）→ 沿用。
	gotVer, gotAt = ResolveClientVersion("9.9.9.9_20991231", def+1)
	if gotVer != "9.9.9.9_20991231" || gotAt != def+1 {
		t.Fatalf("newer saved: got (%q,%d)", gotVer, gotAt)
	}
	// 空保存值 → 默认。
	gotVer, _ = ResolveClientVersion("  ", def+1)
	if gotVer != defaultClientVer {
		t.Fatalf("empty saved: got %q", gotVer)
	}
}

func TestResolveClientVersionUpdatedAtRules(t *testing.T) {
	const now = int64(1_800_000_000_000)
	// 显式传入优先。
	if got := ResolveClientVersionUpdatedAt("a", "b", 5, 42, now); got != 42 {
		t.Fatalf("requested: got %d", got)
	}
	// 版本变化记 now。
	if got := ResolveClientVersionUpdatedAt("a", "b", 5, 0, now); got != now {
		t.Fatalf("changed: got %d", got)
	}
	// 版本未变保留当前值。
	if got := ResolveClientVersionUpdatedAt("a", "a", 5, 0, now); got != 5 {
		t.Fatalf("unchanged: got %d", got)
	}
	// 版本未变且当前无时间戳 → 默认。
	if got := ResolveClientVersionUpdatedAt("a", "a", 0, 0, now); got != defaultClientVerUpdatedAtMs {
		t.Fatalf("unchanged-no-ts: got %d", got)
	}
	// 前后空格不视为版本变化。
	if got := ResolveClientVersionUpdatedAt(" a ", "a", 7, 0, now); got != 7 {
		t.Fatalf("trim: got %d", got)
	}
}
