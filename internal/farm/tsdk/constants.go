package tsdk

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Runtime constants — mirrored from qq-farm-bot/core/src/utils/tsdk-runtime.ts (v3.9.0).
const (
	TSDKVersion = "v3.9.0.1789137379"
	TSDKSHA256  = "1744e339d43425f9f24834fd49b3239f824f57fe76242d5b3128ac55b3110ac5"
	// MiniProgramAppID is the WX mini-program app id (host profile wx).
	MiniProgramAppID = "wx5306c5978fdb76e4"
	// QQMiniProgramAppID is the QQ mini-program app id (host profile qq).
	QQMiniProgramAppID = "1112386029"
	TSDKGameID         = 3167
	TSDKAppKey         = "0"
	MergedDataKey      = 1871261153
	// QQUserDataPath is the fixed user data path reported for QQ hosts.
	QQUserDataPath = "qqfile://usr/"
	// QQDeviceText is the fixed device text reported for QQ hosts.
	QQDeviceText = "windows;windows;windows 10.0;0;"
)

// RuntimeTable is the fixed host table written by import a.k.
var RuntimeTable = []byte{
	93, 86, 110, 34, 65, 129, 8, 113, 53, 192, 121, 32, 86, 162, 255, 139,
	217, 70, 223, 0, 45, 176, 85, 103, 234, 116, 120, 194, 206, 7, 176, 222,
	56, 6, 161, 159, 154, 231, 93, 229, 39, 107, 197, 136, 167, 52, 155, 228,
	209, 117, 218, 8, 107, 241, 32, 62, 53, 200, 238,
}

// MergedDataSegments holds the metadata segment decrypted before
// decrypt_all_data/x/G (bot MERGED_DATA_METADATA).
var MergedDataSegments = [][2]uint32{
	{67371008, 404},
}

// QQHostFeatureState pins the byte-level "host feature state" layout verified
// against the QQ host; used by normalizeQqHostFeatureState.
var QQHostFeatureState = struct {
	CurrentPtr  uint32
	ReferencePtr uint32
	Length      uint32
	NodeMismatchIndex uint32
}{
	CurrentPtr:       17288,
	ReferencePtr:     17352,
	Length:           64,
	NodeMismatchIndex: 1,
}

// VerifyWASMHash checks the on-disk tsdk.wasm against the expected SHA-256.
func VerifyWASMHash(wasm []byte) error {
	sum := sha256.Sum256(wasm)
	got := hex.EncodeToString(sum[:])
	if got != TSDKSHA256 {
		return fmt.Errorf("TSDK file hash mismatch: got %s want %s", got, TSDKSHA256)
	}
	return nil
}
