package logic_test

import (
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
)

func TestGainDisplayNameCurrencies(t *testing.T) {
	cases := map[int64]string{
		1: "金币", 1001: "金币",
		2: "经验", 1101: "经验",
		1002: "点券",
		1005: "金豆",
	}
	for id, want := range cases {
		if got := logic.GainDisplayName(id); got != want {
			t.Fatalf("GainDisplayName(%d)=%q want %q", id, got, want)
		}
	}
}

func TestGainDisplayNameFallbackWithoutConfig(t *testing.T) {
	// 无目录数据时兜底 物品#ID（rust gain_display_name 尾分支）。
	if got := logic.GainDisplayName(999999999); got != "物品#999999999" {
		t.Fatalf("fallback=%q", got)
	}
}

func TestFormatGainsJoinsWithDunhao(t *testing.T) {
	got := logic.FormatGains([]logic.GainEntry{{ID: 1001, Count: 100}, {ID: 1002, Count: 3}})
	if got != "金币×100、点券×3" {
		t.Fatalf("FormatGains=%q", got)
	}
	if logic.FormatGains(nil) != "" {
		t.Fatal("empty entries should render empty text")
	}
}

func TestAggregateGainsSumsByIDSorted(t *testing.T) {
	// rust to_entries 只按 count>0 聚合，不做 id 过滤；输出按 id 升序。
	items := []*corepb.Item{
		{Id: 1002, Count: 3},
		{Id: 1001, Count: 50},
		{Id: 1002, Count: 4},
		{Id: 1001, Count: 0},
		{Id: 0, Count: 9},
	}
	got := logic.AggregateGains(items)
	if len(got) != 3 {
		t.Fatalf("aggregate=%+v", got)
	}
	if got[0].ID != 0 || got[0].Count != 9 {
		t.Fatalf("id0 row=%+v", got[0])
	}
	if got[1].ID != 1001 || got[1].Count != 50 {
		t.Fatalf("gold row=%+v", got[1])
	}
	if got[2].ID != 1002 || got[2].Count != 7 {
		t.Fatalf("ticket row=%+v", got[2])
	}
}
