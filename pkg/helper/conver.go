package helper

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cast"
)

type KeyExtractor func(interface{}) uint64

// ConvertSlice2MapIdKey slice 转换为map ，key采用通用的id
func ConvertSlice2MapIdKey(slice interface{}, keyExtractor KeyExtractor) map[uint64]interface{} {
	// 反射获取反射类型
	v := reflect.ValueOf(slice)
	// 确保传入的是一个切片
	if v.Kind() != reflect.Slice {
		panic("ConvertSliceToMap expects a slice")
	}

	// 创建map
	m := make(map[uint64]interface{})
	for i := 0; i < v.Len(); i++ {
		// 从切片中获取单个元素
		item := v.Index(i).Interface()
		// 使用键提取函数获取键
		key := keyExtractor(item)
		// 将元素添加到map中
		m[key] = item
	}
	return m
}

// StructToMap 使用反射将结构体转换为 map
func StructToMap(obj interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	v := reflect.ValueOf(obj)
	// 如果传入的是指针，获取指针指向的值
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	// 确保传入的是结构体
	if v.Kind() != reflect.Struct {
		return result
	}
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := typ.Field(i)
		// 获取结构体字段名
		key := field.Tag.Get("json")
		if key == "" {
			// 如果 json tag 为空，则使用字段名作为键名
			key = field.Name
		}
		// 获取结构体字段的值
		value := v.Field(i).Interface()
		result[key] = value
	}
	return result
}

// String2Int 将数组的string转int
func String2Int(strArr []string) []int {
	res := make([]int, len(strArr))
	for index, val := range strArr {
		res[index], _ = strconv.Atoi(val)
	}
	return res
}

// ToCamelCase 将字符串转换成驼峰写法
func ToCamelCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '_' || r == '-' || r == '.' || r == '~' || r == ':'
	})
	var result string
	for i, word := range words {
		if i == 0 {
			result += strings.ToLower(word)
		} else {
			result += strings.Title(word)
		}
	}
	return result
}

// ArrayChunk 数组分组
func ArrayChunk[T any](arr []T, size int) [][]T {
	var chunks [][]T
	for i := 0; i < len(arr); i += size {
		end := i + size
		if end > len(arr) {
			end = len(arr)
		}
		chunks = append(chunks, arr[i:end])
	}
	return chunks
}

// IntsToString 将int切片转换成字符串
func IntsToString[T any](numbers []T, sep string) string {
	strNumbers := make([]string, len(numbers))
	for i, num := range numbers {
		strNumbers[i] = cast.ToString(num)
	}
	return strings.Join(strNumbers, sep)
}

// AnyToInts 将任何类型的切片转化成int切片
func AnyToInts[T comparable](numbers []T) []int {
	intNumbers := make([]int, len(numbers))
	for i, num := range numbers {
		intNumbers[i] = cast.ToInt(num)
	}
	return intNumbers
}

// AnyToUint64 将任何类型的切片转化成uin64t切片
func AnyToUint64[T comparable](numbers []T) []uint64 {
	intNumbers := make([]uint64, len(numbers))
	for i, num := range numbers {
		intNumbers[i] = cast.ToUint64(num)
	}
	return intNumbers
}

// ToPointer 将任何类型转化成指针类型
func ToPointer[T comparable](s T) *T {
	return &s
}
