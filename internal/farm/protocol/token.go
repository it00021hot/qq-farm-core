package protocol

import (
	"crypto/rand"
	"math/big"
)

const tokenAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// CreateGatewayToken generates a 65–128 char alphanumeric token ending with '='.
// Matches qq-farm-bot/core/src/utils/gateway-token.ts
func CreateGatewayToken() string {
	n, err := rand.Int(rand.Reader, big.NewInt(64))
	if err != nil {
		n = big.NewInt(0)
	}
	length := 64 + int(n.Int64())
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		for i := range buf {
			buf[i] = byte(i)
		}
	}
	out := make([]byte, length+1)
	for i := 0; i < length; i++ {
		out[i] = tokenAlphabet[int(buf[i])%len(tokenAlphabet)]
	}
	out[length] = '='
	return string(out)
}
