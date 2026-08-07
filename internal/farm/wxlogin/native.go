package wxlogin

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	recordVersion = 0xF103
	hostApp       = "wxd44977328b36e647"
	transferPath  = "/ilink/ilinkapp/mp/wxaruntime_transfer"
	transferHost  = "shortcloud.weixin.com"
)

var serverPub = mustHex("04ef87876d6478b15f1796eab12068610541173b7176b67f1dcc86683e901acd44d18b4ac36938251d0812dd0cf842aa2d6cbb8115712d1c0087dcefc14a44cd58")

type target struct {
	IP   string
	Port int
}

type nativeSession struct {
	SendKey, RecvKey, F9 []byte
	UIN                  uint64
	DeviceID, HostAppID  []byte
	PSK, Ticket          []byte
}

type hybridTemp struct {
	Key  *ecdh.PrivateKey
	OKM  []byte
	Comp []byte
}

type record struct {
	Type byte
	Body []byte
}

func mustHex(value string) []byte {
	data, err := hex.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return data
}

func vi(n uint64) []byte {
	out := make([]byte, 0, 10)
	for {
		b := byte(n & 127)
		n >>= 7
		if n != 0 {
			b |= 128
		}
		out = append(out, b)
		if n == 0 {
			return out
		}
	}
}

func pbl(field int, value []byte) []byte {
	out := append(vi(uint64(field*8+2)), vi(uint64(len(value)))...)
	return append(out, value...)
}

func pbv(field int, value uint64) []byte {
	return append(vi(uint64(field*8)), vi(value)...)
}

func rvi(data []byte, offset int) (uint64, int, error) {
	var n uint64
	for shift := uint(0); offset < len(data); shift += 7 {
		x := data[offset]
		offset++
		if shift >= 64 && x != 0 {
			return 0, 0, fmt.Errorf("varint overflow")
		}
		n |= uint64(x&127) << shift
		if x&128 == 0 {
			return n, offset, nil
		}
	}
	return 0, 0, fmt.Errorf("truncated varint")
}

type pbValue struct {
	Bytes []byte
	Int   uint64
	IsInt bool
}

func pbf(data []byte) (map[int]pbValue, error) {
	out := make(map[int]pbValue)
	for offset := 0; offset < len(data); {
		tag, next, err := rvi(data, offset)
		if err != nil {
			return nil, err
		}
		offset = next
		field := int(tag >> 3)
		switch tag & 7 {
		case 0:
			value, end, err := rvi(data, offset)
			if err != nil {
				return nil, err
			}
			out[field] = pbValue{Int: value, IsInt: true}
			offset = end
		case 2:
			length, start, err := rvi(data, offset)
			if err != nil || length > uint64(len(data)-start) {
				return nil, fmt.Errorf("truncated protobuf")
			}
			end := start + int(length)
			out[field] = pbValue{Bytes: data[start:end]}
			offset = end
		default:
			return out, nil
		}
	}
	return out, nil
}

func requiredField(fields map[int]pbValue, field int, name string) ([]byte, error) {
	value, ok := fields[field]
	if !ok || value.IsInt {
		keys := make([]int, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		sort.Ints(keys)
		return nil, fmt.Errorf("%s is missing (fields: %v)", name, keys)
	}
	return value.Bytes, nil
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

func expand(secret []byte, label string, context []byte, size int) []byte {
	out := make([]byte, 0, size+32)
	var previous []byte
	for counter := byte(1); len(out) < size; counter++ {
		data := make([]byte, 0, len(previous)+len(label)+len(context)+1)
		data = append(data, previous...)
		data = append(data, label...)
		data = append(data, context...)
		data = append(data, counter)
		previous = hmacSHA256(secret, data)
		out = append(out, previous...)
	}
	return out[:size]
}

func nonce(iv []byte, sequence uint64) []byte {
	out := append([]byte(nil), iv...)
	var seq [8]byte
	binary.BigEndian.PutUint64(seq[:], sequence)
	for i := range seq {
		out[len(out)-8+i] ^= seq[i]
	}
	return out
}

func gcm(key, iv []byte, sequence uint64, recordType byte, data []byte, decrypt bool) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	var aad [13]byte
	binary.BigEndian.PutUint64(aad[:8], sequence)
	aad[8] = recordType
	binary.BigEndian.PutUint16(aad[9:11], recordVersion)
	length := len(data) + aead.Overhead()
	if decrypt {
		length = len(data)
	}
	binary.BigEndian.PutUint16(aad[11:13], uint16(length))
	if decrypt {
		return aead.Open(nil, nonce(iv, sequence), data, aad[:])
	}
	return aead.Seal(nil, nonce(iv, sequence), data, aad[:]), nil
}

func layout(key, plain, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	iv := make([]byte, aead.NonceSize())
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	sealed := aead.Seal(nil, iv, plain, aad)
	ciphertext, tag := sealed[:len(sealed)-aead.Overhead()], sealed[len(sealed)-aead.Overhead():]
	out := append([]byte(nil), ciphertext...)
	out = append(out, iv...)
	return append(out, tag...), nil
}

func unlayout(key, blob, aad []byte) ([]byte, error) {
	if len(blob) < 28 {
		return nil, fmt.Errorf("invalid encrypted layout")
	}
	split := len(blob) - 28
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	sealed := append([]byte(nil), blob[:split]...)
	sealed = append(sealed, blob[split+12:]...)
	return aead.Open(nil, blob[split:split+12], sealed, aad)
}

func makeRecord(recordType byte, body []byte) []byte {
	out := make([]byte, 5+len(body))
	out[0] = recordType
	binary.BigEndian.PutUint16(out[1:3], recordVersion)
	binary.BigEndian.PutUint16(out[3:5], uint16(len(body)))
	copy(out[5:], body)
	return out
}

func parseRecords(data []byte) []record {
	var out []record
	for offset := 0; offset+5 <= len(data); {
		length := int(binary.BigEndian.Uint16(data[offset+3 : offset+5]))
		if binary.BigEndian.Uint16(data[offset+1:offset+3]) != recordVersion || offset+5+length > len(data) {
			break
		}
		out = append(out, record{data[offset], data[offset+5 : offset+5+length]})
		offset += 5 + length
	}
	return out
}

func handshake(handshakeType byte, body []byte) []byte {
	out := make([]byte, 5+len(body))
	binary.BigEndian.PutUint32(out[:4], uint32(len(body)+1))
	out[4] = handshakeType
	copy(out[5:], body)
	return out
}

func splitHandshake(data []byte) (byte, []byte, error) {
	if len(data) < 5 || int(binary.BigEndian.Uint32(data[:4]))+4 > len(data) {
		return 0, nil, fmt.Errorf("invalid handshake")
	}
	return data[4], data[5:], nil
}

func lz4Literal(data []byte) []byte {
	if len(data) < 15 {
		return append([]byte{byte(len(data) << 4)}, data...)
	}
	out := []byte{0xF0}
	for n := len(data) - 15; n >= 255; n -= 255 {
		out = append(out, 255)
	}
	out = append(out, byte((len(data)-15)%255))
	return append(out, data...)
}

func lz4(data []byte) ([]byte, error) {
	out := make([]byte, 0)
	for i := 0; i < len(data); {
		token := data[i]
		i++
		n := int(token >> 4)
		if n == 15 {
			for {
				if i >= len(data) {
					return nil, fmt.Errorf("invalid lz4 literal")
				}
				x := int(data[i])
				i++
				n += x
				if x != 255 {
					break
				}
			}
		}
		if i+n > len(data) {
			return nil, fmt.Errorf("invalid lz4 literal length")
		}
		out = append(out, data[i:i+n]...)
		i += n
		if i >= len(data) {
			break
		}
		if i+2 > len(data) {
			return nil, fmt.Errorf("invalid lz4 offset")
		}
		offset := int(binary.LittleEndian.Uint16(data[i : i+2]))
		i += 2
		if offset == 0 || offset > len(out) {
			return nil, fmt.Errorf("invalid lz4 offset")
		}
		match := int(token&15) + 4
		if token&15 == 15 {
			for {
				if i >= len(data) {
					return nil, fmt.Errorf("invalid lz4 match")
				}
				x := int(data[i])
				i++
				match += x
				if x != 255 {
					break
				}
			}
		}
		for j := 0; j < match; j++ {
			out = append(out, out[len(out)-offset])
		}
	}
	return out, nil
}

func wpkg(ints map[int]uint64, byteFields map[int][]byte) []byte {
	keys := make([]int, 0, len(ints))
	for key := range ints {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	payload := vi(1)
	for _, key := range keys {
		payload = append(payload, vi(uint64(key))...)
		payload = append(payload, vi(ints[key])...)
	}
	payload = append(payload, vi(0)...)
	keys = keys[:0]
	for key := range byteFields {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	for _, key := range keys {
		payload = append(payload, vi(uint64(key))...)
		payload = append(payload, vi(uint64(len(byteFields[key])))...)
		payload = append(payload, byteFields[key]...)
	}
	out := append(payload, vi(0)...)
	return append(out, vi(uint64(len(payload)+1))...)
}

func readWpkg(data []byte) (int, error) {
	_, offset, err := rvi(data, 0)
	if err != nil {
		return 0, err
	}
	for {
		field, next, err := rvi(data, offset)
		if err != nil {
			return 0, err
		}
		offset = next
		if field == 0 {
			break
		}
		_, offset, err = rvi(data, offset)
		if err != nil {
			return 0, err
		}
	}
	for {
		field, next, err := rvi(data, offset)
		if err != nil {
			return 0, err
		}
		offset = next
		if field == 0 {
			break
		}
		length, start, err := rvi(data, offset)
		if err != nil || length > uint64(len(data)-start) {
			return 0, fmt.Errorf("invalid wpkg")
		}
		offset = start + int(length)
	}
	_, offset, err = rvi(data, offset)
	return offset, err
}

func shortPacket(command, sequence uint32, body []byte) []byte {
	out := make([]byte, 16+len(body))
	binary.BigEndian.PutUint32(out[:4], uint32(len(out)))
	binary.BigEndian.PutUint16(out[4:6], 0x1110)
	binary.BigEndian.PutUint16(out[6:8], 0x076D)
	binary.BigEndian.PutUint32(out[8:12], command)
	binary.BigEndian.PutUint32(out[12:16], sequence)
	copy(out[16:], body)
	return out
}

func parseShort(data []byte) (uint32, []byte, error) {
	if len(data) < 16 {
		return 0, nil, fmt.Errorf("invalid shortlink")
	}
	length := int(binary.BigEndian.Uint32(data[:4]))
	if length > len(data) {
		return 0, nil, fmt.Errorf("invalid shortlink")
	}
	return binary.BigEndian.Uint32(data[8:12]), data[16:length], nil
}

func newECDH() (*ecdh.PrivateKey, []byte, error) {
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return key, key.PublicKey().Bytes(), nil
}

func clientHello(pub1, pub2 []byte) ([]byte, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return nil, err
	}
	base := append([]byte{3, 0xF1, 1, 0xC0, 0x2B}, random...)
	timestamp := make([]byte, 4)
	binary.BigEndian.PutUint32(timestamp, uint32(time.Now().Unix()))
	base = append(base, timestamp...)
	offers := make([]byte, 0)
	for i, pub := range [][]byte{pub1, pub2} {
		share := make([]byte, 6)
		binary.BigEndian.PutUint32(share[:4], uint32(i+1))
		binary.BigEndian.PutUint16(share[4:6], 65)
		share = append(share, pub...)
		length := make([]byte, 4)
		binary.BigEndian.PutUint32(length, uint32(len(share)))
		offers = append(offers, length...)
		offers = append(offers, share...)
	}
	keyShares := append([]byte{0, 16, 2}, offers...)
	keyShares = append(keyShares, 0, 0, 0, 1)
	extension := []byte{1, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(extension[1:5], uint32(len(keyShares)))
	extension = append(extension, keyShares...)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(extension)))
	return handshake(1, append(append(base, length...), extension...)), nil
}

func pskClientHello(ticket []byte, timestamp uint32) ([]byte, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return nil, err
	}
	u32 := func(value uint32) []byte {
		out := make([]byte, 4)
		binary.BigEndian.PutUint32(out, value)
		return out
	}
	ticketExtension := append([]byte{0, 0x0F, 1}, u32(uint32(len(ticket)))...)
	ticketExtension = append(ticketExtension, ticket...)
	extension := append([]byte{1}, u32(uint32(len(ticketExtension)))...)
	extension = append(extension, ticketExtension...)
	body := append([]byte{3, 0xF1, 1, 0, 0xA8}, random...)
	body = append(body, u32(timestamp)...)
	body = append(body, u32(uint32(len(extension)))...)
	body = append(body, extension...)
	return handshake(1, body), nil
}

type trafficKeys struct {
	ClientKey, ServerKey []byte
	ClientIV, ServerIV   []byte
}

func keys(secret []byte, label string, hash []byte) trafficKeys {
	value := expand(secret, label, hash, 56)
	return trafficKeys{value[:16], value[16:32], value[32:44], value[44:56]}
}

func oneWayKeys(secret []byte, label string, hash []byte) ([]byte, []byte) {
	value := expand(secret, label, hash, 28)
	return value[:16], value[16:28]
}

func manualRequest(loginBuffer string, app []byte) ([]byte, []byte, []byte, error) {
	raw, err := base64.StdEncoding.DecodeString(loginBuffer)
	if err != nil {
		return nil, nil, nil, err
	}
	fields, err := pbf(raw)
	if err != nil {
		return nil, nil, nil, err
	}
	ticket, okTicket := fields[1]
	device, okDevice := fields[2]
	if !okTicket || ticket.IsInt || len(ticket.Bytes) == 0 || !okDevice || device.IsInt || len(device.Bytes) == 0 {
		return nil, nil, nil, fmt.Errorf("invalid login buffer")
	}
	host := []byte(hostApp)
	if value, ok := fields[3]; ok && !value.IsInt && len(value.Bytes) != 0 {
		host = value.Bytes
	}
	base := append(pbl(1, app), pbv(2, 1901)...)
	request := pbl(1, base)
	request = append(request, pbl(3, pbl(1, ticket.Bytes))...)
	request = append(request, pbv(4, 4)...)
	request = append(request, pbl(6, nil)...)
	request = append(request, pbv(7, 0)...)
	request = append(request, pbv(8, 6)...)
	return request, device.Bytes, host, nil
}

func hybrid(plain []byte) (*hybridTemp, []byte, error) {
	key, public, err := newECDH()
	if err != nil {
		return nil, nil, err
	}
	peer, err := ecdh.P256().NewPublicKey(serverPub)
	if err != nil {
		return nil, nil, err
	}
	shared, err := key.ECDH(peer)
	if err != nil {
		return nil, nil, err
	}
	secret := sha256.Sum256(shared)
	h1 := sha256.Sum256(bytes.Join([][]byte{[]byte("1"), []byte("415"), public}, nil))
	cek := make([]byte, 32)
	if _, err := rand.Read(cek); err != nil {
		return nil, nil, err
	}
	encryptedKey, err := layout(secret[:24], cek, h1[:])
	if err != nil {
		return nil, nil, err
	}
	okm := expand(hmacSHA256([]byte("security hdkf expand"), cek), "", h1[:], 56)
	compressed := lz4Literal(plain)
	h2 := sha256.Sum256(bytes.Join([][]byte{[]byte("1"), []byte("415"), public, encryptedKey}, nil))
	encrypted, err := layout(okm[:24], compressed, h2[:])
	if err != nil {
		return nil, nil, err
	}
	keyBlock := append(pbv(1, 415), pbl(2, public)...)
	wire := append(pbv(1, 1), pbl(2, keyBlock)...)
	wire = append(wire, pbl(3, encryptedKey)...)
	wire = append(wire, pbl(4, nil)...)
	wire = append(wire, pbl(5, encrypted)...)
	return &hybridTemp{key, okm, compressed}, wire, nil
}

func parseManual(body []byte, temp *hybridTemp) ([]byte, []byte, []byte, uint64, error) {
	var hybridResponse []byte
	if offset, err := readWpkg(body); err == nil && offset < len(body) && body[offset] == 0x0A {
		hybridResponse = body[offset:]
	}
	if hybridResponse == nil {
		marker := []byte{0x08, 0x9F, 0x03, 0x12, 0x41, 0x04}
		offset := bytes.Index(body, marker)
		if offset < 2 {
			return nil, nil, nil, 0, fmt.Errorf("HybridEcdhResponse not found")
		}
		hybridResponse = body[offset-2:]
	}
	response, err := pbf(hybridResponse)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	keyBlock, err := requiredField(response, 1, "HybridEcdhResponse field 1")
	if err != nil {
		return nil, nil, nil, 0, err
	}
	keyFields, err := pbf(keyBlock)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	peerBytes, err := requiredField(keyFields, 2, "HybridEcdhResponse server public key")
	if err != nil {
		return nil, nil, nil, 0, err
	}
	ciphertext, err := requiredField(response, 3, "HybridEcdhResponse ciphertext")
	if err != nil {
		return nil, nil, nil, 0, err
	}
	credential := uint64(1)
	if value, ok := response[2]; ok && value.IsInt {
		credential = value.Int
	}
	peer, err := ecdh.P256().NewPublicKey(peerBytes)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	shared, err := temp.Key.ECDH(peer)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	secret := sha256.Sum256(shared)
	aadInput := append([]byte(nil), temp.OKM[24:]...)
	aadInput = append(aadInput, temp.Comp...)
	aadInput = append(aadInput, "415"...)
	aadInput = append(aadInput, peerBytes...)
	aadInput = append(aadInput, strconv.FormatUint(credential, 10)...)
	aad := sha256.Sum256(aadInput)
	compressed, err := unlayout(secret[:24], ciphertext, aad[:])
	if err != nil {
		return nil, nil, nil, 0, err
	}
	plain, err := lz4(compressed)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	manual, err := pbf(plain)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	bodyBytes, err := requiredField(manual, 3, "ManualAuthResponse field 3")
	if err != nil {
		return nil, nil, nil, 0, err
	}
	bodyFields, err := pbf(bodyBytes)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	sessionBlock, ok := bodyFields[2]
	if !ok || sessionBlock.IsInt {
		code := "unknown"
		if value, exists := bodyFields[4]; exists && value.IsInt {
			code = strconv.FormatUint(value.Int, 10)
		}
		detail := "unknown error"
		if value, exists := bodyFields[5]; exists && !value.IsInt {
			detail = string(value.Bytes)
		}
		return nil, nil, nil, 0, fmt.Errorf("ManualAuth rejected: code=%s message=%s", code, detail)
	}
	identityBlock, err := requiredField(bodyFields, 3, "ManualAuthResponse identity block")
	if err != nil {
		return nil, nil, nil, 0, err
	}
	sessionFields, err := pbf(sessionBlock.Bytes)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	identityFields, err := pbf(identityBlock)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	sendKey, err := requiredField(sessionFields, 1, "ManualAuthResponse send key")
	if err != nil {
		return nil, nil, nil, 0, err
	}
	recvKey, err := requiredField(sessionFields, 2, "ManualAuthResponse receive key")
	if err != nil {
		return nil, nil, nil, 0, err
	}
	var f9 []byte
	if value, ok := sessionFields[9]; ok && !value.IsInt {
		f9 = value.Bytes
	}
	uin, ok := identityFields[1]
	if !ok || !uin.IsInt {
		return nil, nil, nil, 0, fmt.Errorf("ManualAuthResponse uin is missing")
	}
	return sendKey, recvKey, f9, uin.Int, nil
}

func jsPlain(uin uint64, appID string, host []byte) ([]byte, error) {
	mac := make([]byte, 6)
	if _, err := rand.Read(mac); err != nil {
		return nil, err
	}
	mac[0] = (mac[0] | 2) & 0xFE
	parts := make([]string, len(mac))
	for i, value := range mac {
		parts[i] = fmt.Sprintf("%02X", value)
	}
	device := []byte(strings.Join(parts, "-"))
	uin32 := uint64(uint32(uin))
	info := func(name string) []byte {
		out := pbl(1, []byte("sessionkey"))
		out = append(out, pbv(2, uin32)...)
		out = append(out, pbl(3, device)...)
		out = append(out, pbv(4, 1661404927)...)
		out = append(out, pbl(5, []byte(name))...)
		return append(out, pbv(6, 0)...)
	}
	request := pbl(1, info("UnifiedPCWindows"))
	request = append(request, pbl(2, []byte(appID))...)
	request = append(request, pbv(4, 1)...)
	request = append(request, pbl(5, nil)...)
	request = append(request, pbl(6, nil)...)
	request = append(request, pbv(7, 1)...)
	out := pbl(1, info("Windows"))
	out = append(out, pbl(2, []byte("/cgi-bin/mmbiz-bin/js-login"))...)
	out = append(out, pbl(3, host)...)
	out = append(out, pbv(4, 5)...)
	out = append(out, pbl(5, request)...)
	out = append(out, pbl(6, []byte(appID))...)
	out = append(out, pbv(7, 1029)...)
	out = append(out, pbv(8, 1610627409)...)
	out = append(out, pbl(9, []byte("WindowsxWebPlugin"))...)
	return append(out, pbv(10, 573651281)...), nil
}

func envelope(session *nativeSession, plain []byte) ([]byte, error) {
	encrypted, err := layout(session.SendKey, lz4Literal(plain), nil)
	if err != nil {
		return nil, err
	}
	ints := map[int]uint64{1: 1, 2: session.UIN, 3: 0, 4: 0, 5: 524545, 6: 11, 7: 0, 8: 0, 9: 0, 10: 1, 11: 0, 12: 0, 13: 0, 17: 0, 18: 1, 20: 1504, 21: 0, 22: session.UIN, 23: 0, 25: 16, 26: 4, 28: 1, 29: 1, 30: 0}
	head := wpkg(ints, map[int][]byte{14: {}, 24: session.DeviceID, 27: session.F9})
	inner := shortPacket(0x0B41, 0, append(head, encrypted...))
	body := make([]byte, 2)
	binary.BigEndian.PutUint16(body, uint16(len(transferPath)))
	body = append(body, transferPath...)
	hostLength := make([]byte, 2)
	binary.BigEndian.PutUint16(hostLength, uint16(len(transferHost)))
	body = append(body, hostLength...)
	body = append(body, transferHost...)
	innerLength := make([]byte, 4)
	binary.BigEndian.PutUint32(innerLength, uint32(len(inner)))
	body = append(body, innerLength...)
	body = append(body, inner...)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(body)))
	return append(length, body...), nil
}

type recordConn struct {
	conn   net.Conn
	reader *bufio.Reader
}

func dialRecord(ctx context.Context, target target, timeout time.Duration) (*recordConn, error) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(target.IP, strconv.Itoa(target.Port)))
	if err != nil {
		return nil, err
	}
	return &recordConn{conn: conn, reader: bufio.NewReader(conn)}, nil
}

func (c *recordConn) send(data []byte) error {
	_ = c.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	_, err := c.conn.Write(data)
	return err
}

func (c *recordConn) take() (record, error) {
	_ = c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	header := make([]byte, 5)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return record{}, err
	}
	length := int(binary.BigEndian.Uint16(header[3:5]))
	body := make([]byte, length)
	if _, err := io.ReadFull(c.reader, body); err != nil {
		return record{}, err
	}
	return record{header[0], body}, nil
}

func resolveTargets(ctx context.Context, kind string) []target {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://aedns.weixin.qq.com/cgi-bin/default/getdns?clientversion=0&devicetype=Windows&uin=0&format=json", nil)
	if err == nil {
		request.Header.Set("User-Agent", "MicroMessenger Client")
		response, requestErr := (&http.Client{Timeout: 10 * time.Second}).Do(request)
		if requestErr == nil {
			defer response.Body.Close()
			var data struct {
				DNS struct {
					DomainList []struct {
						Name   string `json:"name"`
						IPList []struct {
							IP string `json:"ip"`
						} `json:"iplist"`
						ProtocolList []struct {
							Name     string `json:"name"`
							PortList []int  `json:"portlist"`
						} `json:"protocollist"`
					} `json:"domainlist"`
				} `json:"dns"`
			}
			if json.NewDecoder(response.Body).Decode(&data) == nil {
				domainName, protocol := "shortcloud.weixin.com", "http"
				if kind == "long" {
					domainName, protocol = "longcloud.weixin.com", "mmtlsovertcp"
				}
				var out []target
				for _, domain := range data.DNS.DomainList {
					if domain.Name != domainName {
						continue
					}
					for _, item := range domain.ProtocolList {
						if item.Name != protocol {
							continue
						}
						for _, ip := range domain.IPList {
							for _, port := range item.PortList {
								out = append(out, target{ip.IP, port})
							}
						}
					}
				}
				if len(out) != 0 {
					return out
				}
			}
		}
	}
	if kind == "long" {
		return []target{{"180.153.202.85", 8080}}
	}
	return []target{{"120.241.131.173", 80}}
}

func establishSession(ctx context.Context, loginBuffer string) (*nativeSession, error) {
	randomApp := make([]byte, 32)
	if _, err := rand.Read(randomApp); err != nil {
		return nil, err
	}
	request, device, host, err := manualRequest(loginBuffer, randomApp)
	if err != nil {
		return nil, err
	}
	var failures []string
	longTargets := resolveTargets(ctx, "long")
	if len(longTargets) > 6 {
		longTargets = longTargets[:6]
	}
	for _, destination := range longTargets {
		session, attemptErr := attemptLong(ctx, destination, request, device, host)
		if attemptErr == nil {
			return session, nil
		}
		message := attemptErr.Error()
		if len(message) > 120 {
			message = message[:120]
		}
		failures = append(failures, fmt.Sprintf("%s:%d %s", destination.IP, destination.Port, message))
	}
	detail := "no HTTPDNS target succeeded"
	if len(failures) != 0 {
		detail = strings.Join(failures, "; ")
	}
	return nil, fmt.Errorf("unable to establish WeChat protocol session: %s", detail)
}

func attemptLong(ctx context.Context, destination target, request, device, host []byte) (*nativeSession, error) {
	connection, err := dialRecord(ctx, destination, 30*time.Second)
	if err != nil {
		return nil, err
	}
	defer connection.conn.Close()
	keyA, pubA, err := newECDH()
	if err != nil {
		return nil, err
	}
	_, pubB, err := newECDH()
	if err != nil {
		return nil, err
	}
	hello, err := clientHello(pubA, pubB)
	if err != nil {
		return nil, err
	}
	if err := connection.send(makeRecord(0x16, hello)); err != nil {
		return nil, err
	}
	serverRecord, err := connection.take()
	if err != nil {
		return nil, err
	}
	_, serverHello, err := splitHandshake(serverRecord.Body)
	if err != nil || len(serverHello) < 40 {
		return nil, fmt.Errorf("invalid ServerHello")
	}
	extensionLength := int(binary.BigEndian.Uint32(serverHello[36:40]))
	if extensionLength > len(serverHello)-40 {
		return nil, fmt.Errorf("invalid ServerHello extension length")
	}
	extension := serverHello[40 : 40+extensionLength]
	if len(extension) < 78 || extension[0] != 1 {
		return nil, fmt.Errorf("invalid ServerHello key-share extension")
	}
	peer, err := ecdh.P256().NewPublicKey(extension[13:78])
	if err != nil {
		return nil, err
	}
	shared, err := keyA.ECDH(peer)
	if err != nil {
		return nil, err
	}
	secret := sha256.Sum256(shared)
	transcript := append(append([]byte(nil), hello...), serverRecord.Body...)
	transcriptHash := sha256.Sum256(transcript)
	handshakeKeys := keys(secret[:], "handshake key expansion", transcriptHash[:])
	var certificateHash []byte
	var ticketEntries [][]byte
	rxSequence := uint64(1)
	for {
		encryptedRecord, err := connection.take()
		if err != nil {
			return nil, err
		}
		plain, err := gcm(handshakeKeys.ServerKey, handshakeKeys.ServerIV, rxSequence, encryptedRecord.Type, encryptedRecord.Body, true)
		rxSequence++
		if err != nil {
			return nil, err
		}
		messageType, body, err := splitHandshake(plain)
		if err != nil {
			return nil, err
		}
		if messageType != 0x14 {
			transcript = append(transcript, plain...)
		}
		if messageType == 0x0F {
			sum := sha256.Sum256(transcript)
			certificateHash = append([]byte(nil), sum[:]...)
		}
		if messageType == 0x04 {
			if len(body) < 1 {
				return nil, fmt.Errorf("invalid ticket message")
			}
			offset := 1
			for i := 0; i < int(body[0]); i++ {
				if offset+4 > len(body) {
					return nil, fmt.Errorf("invalid ticket entry")
				}
				length := int(binary.BigEndian.Uint32(body[offset : offset+4]))
				if offset+4+length > len(body) {
					return nil, fmt.Errorf("invalid ticket entry")
				}
				ticketEntries = append(ticketEntries, append([]byte(nil), body[offset+4:offset+4+length]...))
				offset += 4 + length
			}
		}
		if messageType != 0x14 {
			continue
		}
		hash := sha256.Sum256(transcript)
		verify := hmacSHA256(expand(secret[:], "server finished", nil, 32), hash[:])
		if len(body) < 2 || !hmac.Equal(body[2:], verify) {
			return nil, fmt.Errorf("MMTLS server verification failed")
		}
		expanded := expand(secret[:], "expanded secret", hash[:], 32)
		appKeys := keys(expanded, "application data key expansion", hash[:])
		finishBody := append([]byte{0, 32}, hmacSHA256(expand(secret[:], "client finished", nil, 32), hash[:])...)
		finish, err := gcm(handshakeKeys.ClientKey, handshakeKeys.ClientIV, 1, 0x16, handshake(0x14, finishBody), false)
		if err != nil {
			return nil, err
		}
		if err := connection.send(makeRecord(0x16, finish)); err != nil {
			return nil, err
		}
		temp, wire, err := hybrid(request)
		if err != nil {
			return nil, err
		}
		ints := map[int]uint64{1: 1, 2: 0, 3: 0, 4: 0, 5: 524545, 6: 11, 7: 0, 8: 0, 9: 0, 10: 1, 11: 0, 12: 0, 13: 0, 17: 0, 18: 1, 20: 1504, 21: 0, 22: 0, 23: 0, 25: 17, 26: 4, 28: 1, 29: 1, 30: 0}
		authBody := append(wpkg(ints, map[int][]byte{14: {}, 24: device, 27: {}}), wire...)
		authEncrypted, err := gcm(appKeys.ClientKey, appKeys.ClientIV, 2, 0x17, shortPacket(0x0D7D, 0, authBody), false)
		if err != nil {
			return nil, err
		}
		if err := connection.send(makeRecord(0x17, authEncrypted)); err != nil {
			return nil, err
		}
		authRecord, err := connection.take()
		if err != nil {
			return nil, err
		}
		authPlain, err := gcm(appKeys.ServerKey, appKeys.ServerIV, rxSequence, authRecord.Type, authRecord.Body, true)
		if err != nil {
			return nil, err
		}
		_, authResponse, err := parseShort(authPlain)
		if err != nil {
			return nil, err
		}
		sendKey, recvKey, f9, uin, err := parseManual(authResponse, temp)
		if err != nil {
			return nil, err
		}
		var ticket []byte
		for _, entry := range ticketEntries {
			if len(entry) != 0 && entry[0] == 1 {
				ticket = entry
				break
			}
		}
		if len(sendKey) == 0 || len(recvKey) == 0 || uin == 0 || len(ticket) == 0 || len(certificateHash) == 0 {
			return nil, fmt.Errorf("ManualAuth did not return a usable session")
		}
		return &nativeSession{sendKey, recvKey, f9, uin, device, host, expand(secret[:], "PSK_ACCESS", certificateHash, 32), ticket}, nil
	}
}

func requestEarly(ctx context.Context, destination target, session *nativeSession, environment []byte) (string, error) {
	timestamp := uint32(time.Now().Unix())
	hello, err := pskClientHello(session.Ticket, timestamp)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(hello)
	earlyKey, earlyIV := oneWayKeys(session.PSK, "early data key expansion", hash[:])
	type8 := []byte{0, 0, 0, 16, 8, 0, 0, 0, 11, 1, 0, 0, 0, 6, 0, 18, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(type8[16:20], timestamp)
	earlyType8, err := gcm(earlyKey, earlyIV, 1, 0x19, type8, false)
	if err != nil {
		return "", err
	}
	earlyEnv, err := gcm(earlyKey, earlyIV, 2, 0x17, environment, false)
	if err != nil {
		return "", err
	}
	alert, err := gcm(earlyKey, earlyIV, 3, 0x15, []byte{0, 0, 0, 3, 0, 1, 1}, false)
	if err != nil {
		return "", err
	}
	body := makeRecord(0x19, hello)
	body = append(body, makeRecord(0x19, earlyType8)...)
	body = append(body, makeRecord(0x17, earlyEnv)...)
	body = append(body, makeRecord(0x15, alert)...)
	requestHead := fmt.Sprintf("POST /mmtls/%08x HTTP/1.0\r\nAccept: */*\r\nCache-Control: no-cache\r\nConnection: close\r\nContent-Length: %d\r\nContent-Type: application/octet-stream\r\nHost: shortcloud.weixin.com\r\nUpgrade: mmtls\r\nUser-Agent: MicroMessenger Client\r\nX-Online-Host: shortcloud.weixin.com\r\n\r\n", timestamp, len(body))
	dialer := net.Dialer{Timeout: 8 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(destination.IP, strconv.Itoa(destination.Port)))
	if err != nil {
		return "", err
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(8 * time.Second))
	if _, err := connection.Write(append([]byte(requestHead), body...)); err != nil {
		return "", err
	}
	raw, err := io.ReadAll(connection)
	if err != nil {
		if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() || len(raw) == 0 {
			return "", err
		}
	}
	headerEnd := bytes.Index(raw, []byte("\r\n\r\n"))
	if headerEnd < 0 {
		return "", fmt.Errorf("ShortLink returned an invalid HTTP response")
	}
	responseRecords := parseRecords(raw[headerEnd+4:])
	var serverHello, appData []byte
	for _, item := range responseRecords {
		if item.Type == 0x16 && serverHello == nil {
			serverHello = item.Body
		}
		if item.Type == 0x17 && appData == nil {
			appData = item.Body
		}
	}
	if serverHello == nil || appData == nil {
		return "", fmt.Errorf("ShortLink response missing ServerHello/AppData")
	}
	transcripts := [][]byte{
		append(append([]byte(nil), hello...), serverHello...),
		bytes.Join([][]byte{hello, type8, serverHello}, nil),
		bytes.Join([][]byte{hello, serverHello, type8}, nil),
	}
	for _, transcript := range transcripts {
		sum := sha256.Sum256(transcript)
		handshakeKey, handshakeIV := oneWayKeys(session.PSK, "handshake key expansion", sum[:])
		for _, sequence := range []uint64{2, 1, 3} {
			decrypted, err := gcm(handshakeKey, handshakeIV, sequence, 0x17, appData, true)
			if err != nil {
				continue
			}
			candidates := [][]byte{decrypted}
			if _, parsed, err := parseShort(decrypted); err == nil {
				candidates = append([][]byte{parsed}, candidates...)
			}
			for _, candidate := range candidates {
				limit := min(220, len(candidate))
				for offset := 0; offset < limit; offset++ {
					compressed, err := unlayout(session.RecvKey, candidate[offset:], nil)
					if err != nil {
						continue
					}
					plain, err := lz4(compressed)
					if err != nil {
						continue
					}
					outer, err := pbf(plain)
					if err != nil {
						continue
					}
					innerBytes, err := requiredField(outer, 2, "wx.login response field 2")
					if err != nil {
						continue
					}
					inner, err := pbf(innerBytes)
					if err != nil {
						continue
					}
					if code, ok := inner[3]; ok && !code.IsInt && len(code.Bytes) != 0 {
						return string(code.Bytes), nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("ShortLink AppData decrypt/parse failed")
}

func getNativeWxLoginCode(ctx context.Context, loginBuffer, appID string) (string, error) {
	session, err := establishSession(ctx, loginBuffer)
	if err != nil {
		return "", err
	}
	plain, err := jsPlain(session.UIN, appID, session.HostAppID)
	if err != nil {
		return "", err
	}
	environment, err := envelope(session, plain)
	if err != nil {
		return "", err
	}
	var lastErr error
	for _, destination := range resolveTargets(ctx, "short") {
		code, err := requestEarly(ctx, destination, session, environment)
		if err == nil {
			return code, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("unable to request wx.login code")
}
