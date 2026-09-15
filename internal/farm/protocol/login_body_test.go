package protocol

import (
	"encoding/hex"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/userpb"
)

const goldenVersion = "1.14.0.4_20260911"

func hexEncode(bytes []byte) string { return hex.EncodeToString(bytes) }

// 官方向量：2026-09-11 QQ 小程序抓包 ws_00001_SEND.bin 解密后的 Login 体，
// 钉死在 bot network-keepalive.test.js / rust login_body.rs。
func TestLoginBodyMatchesOfficialCaptureByteForByte(t *testing.T) {
	body := BuildLoginBody(goldenVersion, "Windows")
	if len(body) != 73 {
		t.Fatalf("login body len = %d, want 73", len(body))
	}
	const want = "180022002a1c0a11312e31342e302e345f3230323630393131120757696e646f7773" +
		"30003a073132333435363742180a0012001a0022002a086f746865722d717130023a0042004a00"
	if got := hexEncode(body); got != want {
		t.Fatalf("login body hex mismatch:\n got %s\nwant %s", got, want)
	}
}

// 官方向量：ws_00114_SEND.bin，gid = 1220537209。
func TestHeartbeatBodyMatchesOfficialCaptureByteForByte(t *testing.T) {
	body := BuildHeartbeatBody(1_220_537_209, goldenVersion)
	if len(body) != 27 {
		t.Fatalf("heartbeat body len = %d, want 27", len(body))
	}
	const want = "08f9d6ffc5041211312e31342e302e345f32303236303931311800"
	if got := hexEncode(body); got != want {
		t.Fatalf("heartbeat body hex mismatch:\n got %s\nwant %s", got, want)
	}
}

// 守卫：proto.Marshal 会省略显式默认值字段（产不出官方向量），
// 防止将来有人「简化」回 proto.Marshal 而破坏字节对齐时能被测试拦住。
func TestProtoMarshalCannotReproduceOfficialLoginBody(t *testing.T) {
	req := &userpb.LoginRequest{
		DeviceInfo: &userpb.DeviceInfo{
			ClientVersion: goldenVersion,
			SysSoftware:   "Windows",
		},
		SceneId: "1234567",
		ReportData: &userpb.ReportData{
			MinigameChannel: "other-qq",
			MinigamePlatid:  2,
		},
	}
	prostBody, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(prostBody) == 73 {
		t.Fatalf("proto3 marshal 不应产出 73 字节的官方向量（会省略默认值字段）")
	}
	if hexEncode(prostBody) == hexEncode(BuildLoginBody(goldenVersion, "Windows")) {
		t.Fatalf("proto.Marshal 不应复现官方 Login 体")
	}
}

// 非默认入参（超长版本号 / 空系统名）仍须产出可解码的同构消息。
func TestBuildersScaleWithArguments(t *testing.T) {
	longVersion := "1.14.0.5_20270101"
	login := BuildLoginBody(longVersion, "Windows Unknown x64")
	decoded := &userpb.LoginRequest{}
	if err := proto.Unmarshal(login, decoded); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if decoded.GetDeviceInfo().GetClientVersion() != longVersion ||
		decoded.GetDeviceInfo().GetSysSoftware() != "Windows Unknown x64" ||
		decoded.GetSceneId() != "1234567" ||
		decoded.GetReportData().GetMinigameChannel() != "other-qq" ||
		decoded.GetReportData().GetMinigamePlatid() != 2 {
		t.Fatalf("login round-trip mismatch: %+v", decoded)
	}

	hb := BuildHeartbeatBody(1, longVersion)
	hbDecoded := &userpb.HeartbeatRequest{}
	if err := proto.Unmarshal(hb, hbDecoded); err != nil {
		t.Fatalf("decode heartbeat: %v", err)
	}
	if hbDecoded.GetGid() != 1 ||
		hbDecoded.GetClientVersion() != longVersion ||
		hbDecoded.GetField_3() != 0 {
		t.Fatalf("heartbeat round-trip mismatch: %+v", hbDecoded)
	}
}
