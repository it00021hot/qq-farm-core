package tsdk

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Runtime constants — see qq-farm-bot/core/docs/tsdk-runtime.md
const (
	TSDKVersion      = "v3.8.6.1785239995"
	TSDKSHA256       = "14754428297ee0d5aa6cceee76e6ef076bdac31ceda0ea2e2bf4a0472c8e717f"
	MiniProgramAppID = "wx5306c5978fdb76e4"
	TSDKGameID       = 3167
	TSDKAppKey       = "0"
	MergedDataKey    = 1871261153
)

// RuntimeTable is the fixed host table written by import a.k.
var RuntimeTable = []byte{
	93, 86, 110, 34, 65, 129, 8, 113, 53, 192, 121, 32, 86, 162, 255, 139,
	217, 70, 223, 0, 45, 176, 85, 103, 234, 116, 120, 194, 206, 7, 176, 222,
	56, 6, 161, 159, 154, 231, 93, 229, 39, 107, 197, 136, 167, 52, 155, 228,
	209, 117, 218, 8, 107, 241, 32, 62, 53, 200, 238,
}

// MergedDataSegments are (ptr, length) pairs decrypted before calling x()/G().
//
// Init order (from tsdk-runtime.md):
//  1. Instantiate WASM with host imports a.a–a.v
//  2. Decrypt 17 mergewasm segments via __mergewasm_shared____wasm_decrypt_strings
//  3. Call export x()  (do NOT call decrypt_all_data — breaks function table)
//  4. Call export G(3167, appKeyPtr)
var MergedDataSegments = [][2]uint32{
	{1024, 5541}, {6580, 8989}, {15585, 33}, {15643, 1}, {15655, 21},
	{15701, 1}, {15713, 21}, {15759, 1}, {15771, 30}, {15826, 14},
	{15875, 1}, {15887, 21}, {15933, 1}, {15945, 671}, {16632, 400},
	{17040, 103}, {67371008, 404},
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
