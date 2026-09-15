package runtime

// BasicNotify 显式 0 值探测（对齐 rust network/notify.rs 的 nested_field_present /
// field_present / find_length_delimited_field 等手写 wire tag 扫描）：
//
// proto3 缺省值就是 0，解码后无法区分「显式 gold=0」和「未携带 gold」。原 bot
// 按 hasOwn(notify.basic, 'gold'|'exp'|'level') 判断，rust 移植了同款 wire tag
// 扫描；这里做 go 侧等价实现，供 applyBasicNotify 用字段存在性代替 >0 判断。
//
// 字段号以 go 侧 BasicNotify proto 定义为准（userpb.pb.go）：
//   - BasicNotify.basic = field 1（length-delimited）
//   - BasicInfo：3=level、4=exp、5=gold（varint）

// basicNotifyFieldPresent 扫描 BasicNotify 的 body 字节，判断 basic 子消息
// （outerField 号，length-delimited）里是否真的携带了 innerField 号字段。
func basicNotifyFieldPresent(buf []byte, outerField, innerField uint64) bool {
	nested, ok := protoLengthDelimitedField(buf, outerField)
	if !ok {
		return false
	}
	return protoFieldPresent(nested, innerField)
}

// protoFieldPresent 顺序扫描 protobuf 二进制，判断指定字段号的 tag 是否出现。
func protoFieldPresent(buf []byte, field uint64) bool {
	i := 0
	for i < len(buf) {
		num, wire, ok := readProtoKey(buf, &i)
		if !ok {
			return false
		}
		if num == field {
			return true
		}
		if !skipProtoValue(buf, &i, wire) {
			return false
		}
	}
	return false
}

// protoLengthDelimitedField 返回 buf 中 field 号 length-delimited 子消息的字节。
func protoLengthDelimitedField(buf []byte, field uint64) ([]byte, bool) {
	i := 0
	for i < len(buf) {
		num, wire, ok := readProtoKey(buf, &i)
		if !ok {
			return nil, false
		}
		if num == field && wire == 2 {
			length, ok := readProtoVarint(buf, &i)
			if !ok {
				return nil, false
			}
			if length > uint64(len(buf)-i) {
				return nil, false
			}
			end := i + int(length)
			return buf[i:end], true
		}
		if !skipProtoValue(buf, &i, wire) {
			return nil, false
		}
	}
	return nil, false
}

// readProtoKey 读一个 field tag：返回 (字段号, wire type)。
func readProtoKey(buf []byte, i *int) (num, wire uint64, ok bool) {
	tag, ok := readProtoVarint(buf, i)
	if !ok {
		return 0, 0, false
	}
	return tag >> 3, tag & 7, true
}

// readProtoVarint 读 base-128 varint。
func readProtoVarint(buf []byte, i *int) (uint64, bool) {
	var result uint64
	shift := uint(0)
	for {
		if *i >= len(buf) || shift >= 64 {
			return 0, false
		}
		b := buf[*i]
		*i++
		result |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return result, true
		}
		shift += 7
	}
}

// skipProtoValue 按 wire type 跳过一个字段值（对齐 rust skip_value：
// 0=varint、1=64-bit、2=length-delimited、5=32-bit，其余非法）。
func skipProtoValue(buf []byte, i *int, wire uint64) bool {
	switch wire {
	case 0:
		_, ok := readProtoVarint(buf, i)
		return ok
	case 1:
		*i += 8
		return *i <= len(buf)
	case 2:
		length, ok := readProtoVarint(buf, i)
		if !ok {
			return false
		}
		if length > uint64(len(buf)-*i) {
			return false
		}
		*i += int(length)
		return true
	case 5:
		*i += 4
		return *i <= len(buf)
	default:
		return false
	}
}
