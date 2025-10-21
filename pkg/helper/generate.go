package helper

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand/v2"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/gogf/gf/v2/text/gstr"
	uuid2 "github.com/google/uuid"
	"github.com/hashicorp/go-uuid"
	"github.com/spf13/cast"
	"github.com/thanhpk/randstr"
)

var (
	snowflakeNode *snowflake.Node
	sequence      uint32
	sequenceMutex sync.Mutex
	generatedIDs  map[int64]bool
	idMutex       sync.Mutex
)

func init() {
	localIp, _ := GetLocalIpToInt()
	snowflakeNode, _ = snowflake.NewNode(int64(localIp) % 1023)
	generatedIDs = make(map[int64]bool)
}

// GenerateUUID 生成唯一ID
func GenerateUUID() int64 {
	// 清理已生成的ID记录，防止内存泄漏
	if len(generatedIDs) > 1000000 { // 假设我们只保留最近的100万个ID
		generatedIDs = make(map[int64]bool)
	}
	// 生成基础ID
	id := snowflakeNode.Generate().Int64()

	// 添加序列号
	sequenceMutex.Lock()
	sequence++
	id += int64(sequence)
	sequenceMutex.Unlock()

	// 检查是否重复
	idMutex.Lock()
	if !generatedIDs[id] {
		generatedIDs[id] = true
		idMutex.Unlock()
		return id
	}
	idMutex.Unlock()

	// 如果重复，等待一小段时间后重试
	time.Sleep(time.Millisecond)
	return GenerateUUID()
}

// GenerateBaseSnowId 生成雪花算法ID 废弃
func GenerateBaseSnowId(num int, n *snowflake.Node) string {
	if n == nil {
		localIp, err := GetLocalIpToInt()
		if err != nil {
			return ""
		}
		node, err := snowflake.NewNode(int64(localIp) % 1023)
		if err != nil {
			return ""
		}
		n = node
	}
	id := n.Generate()
	switch num {
	case 2:
		return id.Base2()
	case 32:
		return id.Base32()
	case 36:
		return id.Base36()
	case 58:
		return id.Base58()
	case 64:
		return id.Base64()
	default:
		return cast.ToString(id.Int64())
	}
}

// GenerateUuid 生成随机字符串
func GenerateUuid(size int) string {
	str, err := uuid.GenerateUUID()
	if err != nil {
		return ""
	}
	return gstr.SubStr(str, 0, size)
}

// RandString 随机字符串
func RandString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}
	return string(b)
}

// GenerateRandomString 生成随机字符串
func GenerateRandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(1 * time.Nanosecond) // 确保每个字符都不同
	}
	return string(result)
}

// GeneratePasswordHash 生成密码hash值
func GeneratePasswordHash(password string, salt string) string {
	s := sha256.New()
	io.WriteString(s, password+salt)
	str := fmt.Sprintf("%x", s.Sum(nil))
	return str
}

// GenerateHash 生成md5 hash值
func GenerateHash(str string) string {
	s := md5.New()
	s.Write([]byte(str))
	return hex.EncodeToString(s.Sum(nil))
}

// GenerateCode 随机生成n位验证码
func GenerateCode(length int) string {
	charset := "0123456789"
	code := make([]byte, length)
	for i := 0; i < length; i++ {
		code[i] = charset[rand.IntN(len(charset))]
	}
	return string(code)
}

// GenerateId 生成随机id
func GenerateId() string {
	return uuid2.NewString()
}

func GenerateClientId() string {
	return randstr.Hex(10)
}

// GenerateSeqNo 生成业务流水编号
// 前缀：
// 发票：FP
// 账单：ZD
// 交易：JY
// 提现：TX
// 充值：CZ
// 汇款：HK
// 开通：KT
// 格式：{前缀}{YYMMDDHHmmss}{商家ID(6-8位)}{随机数字(4-6位)}
func GenerateSeqNo(prefix string, mchId uint64) string {
	// 生成时间部分（年份只保留后两位）
	timeStr := time.Now().Format("060102150405")

	// 商家ID部分 10位，不足补0
	mchStr := fmt.Sprintf("%06d", mchId)

	// 生成6位随机数字
	randomNum := fmt.Sprintf("%06d", rand.IntN(1000000))

	return fmt.Sprintf("%s%s%s%s", prefix, timeStr, mchStr, string(randomNum))
}

// 商家开放平台code
func GenerateAppCode(mchId uint64) string {
	// 商家ID部分 10位，不足补0
	mchStr := fmt.Sprintf("%06d", mchId)

	return fmt.Sprintf("%s%s%s", "AC", mchStr, RandString(6))
}

// GenerateAppKeyPair 生成应用的AK和SK密钥对
// 返回值: appKey string, appSecret string
func GenerateAppKeyPair() (string, string) {
	// 生成AppKey: 前缀AK + 32位随机字符(包含数字和大小写字母)
	appKey := fmt.Sprintf("AK%s", RandString(32))

	// 生成AppSecret: 前缀SK + 64位随机字符(包含数字和大小写字母) + 时间戳
	timestamp := time.Now().UnixNano() / 1e6 // 毫秒时间戳
	randomStr := RandString(64)
	rawSecret := fmt.Sprintf("SK%s%d", randomStr, timestamp)

	// 对AppSecret进行SHA256哈希处理，增加安全性
	hash := sha256.New()
	hash.Write([]byte(rawSecret))
	appSecret := hex.EncodeToString(hash.Sum(nil))

	return appKey, appSecret
}

// ValidateAppKeyPair 验证AK/SK密钥对格式是否合法
func ValidateAppKeyPair(appKey string, appSecret string) bool {
	// 验证AppKey格式
	if len(appKey) != 34 || !strings.HasPrefix(appKey, "AK") {
		return false
	}

	// 验证AppSecret格式(SHA256哈希后的字符串长度为64)
	if len(appSecret) != 64 {
		return false
	}

	// 验证字符合法性
	validChars := regexp.MustCompile("^[a-fA-F0-9]+$")
	return validChars.MatchString(appSecret)
}
