package runtime

import (
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/userpb"
	"google.golang.org/protobuf/proto"
)

// --- 手写 wire 字节辅助（proto.Marshal 会省略 proto3 零值，无法构造「显式 0」） ---

func appendProtoVarint(buf []byte, v uint64) []byte {
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	return append(buf, byte(v))
}

func basicVarintField(field uint64, value uint64) []byte {
	buf := appendProtoVarint(nil, (field<<3)|0)
	return appendProtoVarint(buf, value)
}

func basicBytesField(field uint64, payload []byte) []byte {
	buf := appendProtoVarint(nil, (field<<3)|2)
	buf = appendProtoVarint(buf, uint64(len(payload)))
	return append(buf, payload...)
}

func basicNotifyBody(basicFields []byte) []byte {
	return basicBytesField(1, basicFields)
}

func newBasicNotifySession() *Session {
	return &Session{id: "0", playerLevel: 42, playerExp: 555, gold: 777, nick: "旧昵称", avatar: "旧头像"}
}

// TestBasicNotifyProbeFieldNumbers 校验探测函数与 go 侧 userpb proto 字段号一致：
// BasicNotify.basic=1，BasicInfo 3=level、4=exp、5=gold。
func TestBasicNotifyProbeFieldNumbers(t *testing.T) {
	body, err := proto.Marshal(&userpb.BasicNotify{Basic: &userpb.BasicInfo{Level: 5, Exp: 6, Gold: 7, Name: "n"}})
	if err != nil {
		t.Fatal(err)
	}
	if !basicNotifyFieldPresent(body, 1, 3) {
		t.Fatal("level (field 3 in basic/field 1) should be present")
	}
	if !basicNotifyFieldPresent(body, 1, 4) {
		t.Fatal("exp (field 4) should be present")
	}
	if !basicNotifyFieldPresent(body, 1, 5) {
		t.Fatal("gold (field 5) should be present")
	}
	if basicNotifyFieldPresent(body, 1, 6) {
		t.Fatal("openid (field 6) should be absent")
	}
	// 缺 basic 子消息时一律 false。
	if basicNotifyFieldPresent([]byte{}, 1, 3) || basicNotifyFieldPresent(nil, 1, 3) {
		t.Fatal("empty body should not report presence")
	}
}

// TestApplyBasicNotifyExplicitZeroApplied 对齐 rust notify.rs 测试
// basic_notify_*：显式 gold=0 / exp=0 必须应用（金币花光 / 经验清零是合法状态），
// 昵称 / 头像非空仍更新（go 已有行为，不得破坏）。
func TestApplyBasicNotifyExplicitZeroApplied(t *testing.T) {
	s := newBasicNotifySession()
	basicFields := append(append(
		basicVarintField(4, 0),    // exp = 0（显式携带）
		basicVarintField(5, 0)..., // gold = 0（显式携带）
	), basicBytesField(2, []byte("新昵称"))...)
	basicFields = append(basicFields, basicBytesField(7, []byte("http://a/b.png"))...)
	body := basicNotifyBody(basicFields)

	s.applyBasicNotify(body)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gold != 0 {
		t.Fatalf("explicit gold=0 should be applied, got %d", s.gold)
	}
	if s.playerExp != 0 {
		t.Fatalf("explicit exp=0 should be applied, got %d", s.playerExp)
	}
	if s.playerLevel != 42 {
		t.Fatalf("absent level should be untouched, got %d", s.playerLevel)
	}
	if s.nick != "新昵称" {
		t.Fatalf("nick update broken, got %q", s.nick)
	}
	if s.avatar != "http://a/b.png" {
		t.Fatalf("avatar update broken, got %q", s.avatar)
	}
}

// TestApplyBasicNotifyMissingFieldsNotApplied：缺字段不得当作 0 应用。
func TestApplyBasicNotifyMissingFieldsNotApplied(t *testing.T) {
	s := newBasicNotifySession()
	body := basicNotifyBody(basicVarintField(3, 8)) // 只带 level=8

	s.applyBasicNotify(body)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.playerLevel != 8 {
		t.Fatalf("explicit level=8 should be applied, got %d", s.playerLevel)
	}
	if s.playerExp != 555 || s.gold != 777 {
		t.Fatalf("absent exp/gold should be untouched, got exp=%d gold=%d", s.playerExp, s.gold)
	}
}

// TestApplyBasicNotifyExplicitZeroLevelNotApplied：显式 level=0 不应用
// （对齐 rust：has_level && level > 0，等级不允许被推送清零）。
func TestApplyBasicNotifyExplicitZeroLevelNotApplied(t *testing.T) {
	s := newBasicNotifySession()
	body := basicNotifyBody(basicVarintField(3, 0))

	s.applyBasicNotify(body)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.playerLevel != 42 {
		t.Fatalf("explicit level=0 should not be applied, got %d", s.playerLevel)
	}
}

// TestProtoProbeSkipsUnknownFields：探测应正确跳过变长 / 定长未知字段。
func TestProtoProbeSkipsUnknownFields(t *testing.T) {
	// 7 号 string 在前、5 号 varint 在后；再混一个 64-bit 定长（wire 1）。
	buf := basicBytesField(7, []byte("avatar"))
	buf = append(buf, 0x29, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08) // field 5 wire 1（8 字节定长）
	buf = append(buf, basicVarintField(5, 0)...)

	if !protoFieldPresent(buf, 5) {
		t.Fatal("field 5 should be found after skipping unknown fields")
	}
	if !protoFieldPresent(buf, 7) {
		t.Fatal("field 7 should be found")
	}
	if protoFieldPresent(buf, 4) {
		t.Fatal("field 4 should be absent")
	}
	nested, ok := protoLengthDelimitedField(buf, 7)
	if !ok || string(nested) != "avatar" {
		t.Fatalf("length-delimited extraction broken: %q ok=%v", nested, ok)
	}
	// 截断 / 坏 varint 不 panic、返回 false（单个完整 tag 命中即算存在，
	// 与 rust field_present 的「读到即真」一致）。
	if protoFieldPresent([]byte{0xff}, 5) {
		t.Fatal("bad varint should not report presence")
	}
}
