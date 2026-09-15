package protocol

// Login / Heartbeat 请求体构造（逐字节对齐官方客户端抓包）。
//
// 1:1 翻译 qq-farm-bot/core/src/utils/network.ts 的 buildLoginBody() /
// buildHeartbeatBody()（bot 9709bcb，2026-09-14 按 2026-09-11 官方抓包逐字节
// 对齐），与 rust crates/qq-farm-core/src/network/login_body.rs 保持同构。
//
// # 为什么手写字节而不是 proto.Marshal
//
// proto3 编码会省略所有取默认值的字段，但官方客户端**显式写出**了它们：
//   - sharer_id = 0 / share_cfg_id = 0（varint 0）
//   - sharer_open_id（空串）、extra（空 bytes）
//   - report_data 的 6 个空字符串字段（callback / cd_extend_info / click_id /
//     clue_token / req_id / trackid）
//   - Heartbeat.field_3 = 0
//
// protobuf-go 产不出这些「显式默认值」，因此按官方抓包逐字节构造
// （Login 固定 73 字节、Heartbeat 固定 27 字节），并用官方向量做 golden
// 测试钉死。改动本文件前先对照 bot network.ts 的同名函数与
// ws_00001_SEND.bin / ws_00114_SEND.bin 抓包向量。

import "encoding/binary"

// protobuf wire type 0（varint）
const wireVarint = 0
// protobuf wire type 2（length-delimited）
const wireLen = 2

// BuildLoginBody 构造 LoginRequest 请求体。官方向量固定 73 字节
// （client_version="1.14.0.4_20260911"、sys_software="Windows" 时），
// 其余入参按实际长度伸缩。
func BuildLoginBody(clientVersion, sysSoftware string) []byte {
	b := make([]byte, 0, 73)
	// field 3 sharer_id = 0（显式 varint 0）
	b = pushTag(b, 3, wireVarint)
	b = binary.AppendUvarint(b, 0)
	// field 4 sharer_open_id = ""（显式空串）
	b = pushLenDelim(b, 4, nil)
	// field 5 device_info { 1: client_version, 2: sys_software }
	di := make([]byte, 0, len(clientVersion)+len(sysSoftware)+4)
	di = pushLenDelim(di, 1, []byte(clientVersion))
	di = pushLenDelim(di, 2, []byte(sysSoftware))
	b = pushLenDelim(b, 5, di)
	// field 6 share_cfg_id = 0（显式 varint 0）
	b = pushTag(b, 6, wireVarint)
	b = binary.AppendUvarint(b, 0)
	// field 7 scene_id = "1234567"
	b = pushLenDelim(b, 7, []byte("1234567"))
	// field 8 report_data { 1..4 空串, 5 "other-qq", 6 = 2, 7..8 空串 }
	rd := make([]byte, 0, 24)
	for field := int64(1); field <= 4; field++ {
		rd = pushLenDelim(rd, field, nil)
	}
	rd = pushLenDelim(rd, 5, []byte("other-qq"))
	rd = pushTag(rd, 6, wireVarint)
	rd = binary.AppendUvarint(rd, 2)
	rd = pushLenDelim(rd, 7, nil)
	rd = pushLenDelim(rd, 8, nil)
	b = pushLenDelim(b, 8, rd)
	// field 9 extra = ""（显式空 bytes）
	b = pushLenDelim(b, 9, nil)
	return b
}

// BuildHeartbeatBody 构造 HeartbeatRequest 请求体，固定 27 字节
// （field_3 显式写 0，proto.Marshal 会把它省掉）。
func BuildHeartbeatBody(gid int64, clientVersion string) []byte {
	b := make([]byte, 0, 27)
	b = pushTag(b, 1, wireVarint)
	b = binary.AppendUvarint(b, uint64(gid))
	b = pushLenDelim(b, 2, []byte(clientVersion))
	b = pushTag(b, 3, wireVarint)
	b = binary.AppendUvarint(b, 0)
	return b
}

func pushTag(buf []byte, field int64, wireType byte) []byte {
	return binary.AppendUvarint(buf, uint64(field)<<3|uint64(wireType))
}

func pushLenDelim(buf []byte, field int64, payload []byte) []byte {
	buf = pushTag(buf, field, wireLen)
	buf = binary.AppendUvarint(buf, uint64(len(payload)))
	return append(buf, payload...)
}
