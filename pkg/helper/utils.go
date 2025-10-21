package helper

import (
	"fmt"
	"math"
	"os/exec"
	"reflect"
	"regexp"
	"strings"

	"gorm.io/gorm/schema"
)

// InAnySlice 判断某个字符串是否在字符串切片中
func InAnySlice[T comparable](haystack []T, needle T) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

// InAnyMap 判断某个map的值是否存在
func InAnyMap[T comparable](haystack map[string]T, needle T) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

// InAnyKeyMap 判断某个map的键是否存在
func InAnyKeyMap[T comparable, V comparable](haystack map[T]V, needle T) bool {
	if _, ok := haystack[needle]; ok {
		return true
	}
	return false
}

// GetKeyByMap 根据map中的值获取键
func GetKeyByMap[T comparable](m map[string]T, value T) string {
	for key, val := range m {
		if val == value {
			return key
		}
	}
	return ""
}

// GetStructColumnName 获取结构体中的字段名称 _type: 1: 获取tag字段值 2：获取结构体字段值
func GetStructColumnName(s interface{}, _type int) ([]string, error) {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Struct {
		return []string{}, fmt.Errorf("interface is not a struct")
	}
	t := v.Type()
	var fields []string
	for i := 0; i < v.NumField(); i++ {
		var field string
		if _type == 1 {
			field = t.Field(i).Tag.Get("json")
			if field == "" {
				tagSetting := schema.ParseTagSetting(t.Field(i).Tag.Get("gorm"), ";")
				field = tagSetting["COLUMN"]
			}
		} else {
			field = t.Field(i).Name
		}
		fields = append(fields, field)
	}
	return fields, nil
}

// GetProjectModuleName 获取当前项目的module名称
func GetProjectModuleName() string {
	cmd := exec.Command("go", "list", "-m")
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	return strings.Trim(string(output), "\n")
}

// GetDeliveryTime 计算配送时间 配送时间默认3000m以内60分钟，超过1000m增加5分钟，返回秒
func GetDeliveryTime(distance int, startDistance int, startTime int, addDistance int, addTime int) int {
	if startDistance == 0 {
		startDistance = 3000
	}
	if startTime == 0 {
		startTime = 60
	}
	if addDistance == 0 {
		addDistance = 1000
	}
	if addTime == 0 {
		addTime = 5
	}
	if distance < startDistance {
		return startTime * 60
	}
	return (int(math.Ceil(float64(distance-startDistance)/float64(addDistance)))*addTime + startTime) * 60
}

// IsValidPhone 验证手机号格式
func IsValidPhone(phone string) bool {
	// 中国大陆手机号格式验证
	reg := regexp.MustCompile(`^1[3-9]\d{9}$`)
	return reg.MatchString(phone)
}

// IsValidEmail 验证邮箱格式
func IsValidEmail(email string) bool {
	// 邮箱格式验证
	reg := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return reg.MatchString(email)
}
