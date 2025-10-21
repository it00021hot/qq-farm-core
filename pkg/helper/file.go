package helper

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// IsPathExist 判断所给路径文件/文件夹是否存在
func IsPathExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

// MakeMultiDir 调用os.MkdirAll递归创建文件夹
func MakeMultiDir(filePath string) error {
	if !IsPathExist(filePath) {
		return os.MkdirAll(filePath, os.ModePerm)
	}
	return nil
}

// MakeFileOrPath 创建文件/文件夹
func MakeFileOrPath(path string) (*os.File, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// ReadLocalFile 读取本地文件并返回字节数组数据
func ReadLocalFile(filePath string) ([]byte, error) {
	// 判断文件是否存在
	if !IsPathExist(filePath) {
		return nil, os.ErrNotExist
	}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 读取文件内容
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return content, nil
}

// GetUploadFileBytes 获取上传文件的数据流
func GetUploadFileBytes(file *multipart.FileHeader) ([]byte, error) {
	// 打开文件
	open, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer open.Close()

	// 预分配合适大小的buffer以减少内存分配
	buffer := make([]byte, file.Size)

	// 直接读取到预分配的buffer中
	_, err = io.ReadFull(open, buffer)
	if err != nil {
		return nil, err
	}

	return buffer, nil
}

// WriteBytesToFile 将字节数组数据写入本地文件
func WriteBytesToFile(data []byte, filePath string) error {
	// 确保目标目录存在
	if err := MakeMultiDir(filepath.Dir(filePath)); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 创建或打开文件
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 写入数据
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// WriteContentToFile 将文件内容写入指定路径
func WriteContentToFile(file *multipart.FileHeader, filePath string) error {
	// 确保目标目录存在
	if err := MakeMultiDir(filepath.Dir(filePath)); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 创建目标文件
	dst, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	// 打开源文件
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer src.Close()

	// 直接复制文件内容，避免将整个文件读入内存
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// MakeTimeFormatDir 创建时间格式的目录 如：upload/{path}/2023-01-07/
//
//	rootPath	根目录   pathName	子目录名称  timeFormat	时间格式	如：2006-01-02、20060102
func MakeTimeFormatDir(rootPath, pathName, timeFormat string) (string, error) {
	filePath := ""
	if pathName != "" {
		filePath = filepath.Join(filePath, pathName)
	}
	if timeFormat == "" {
		timeFormat = time.DateOnly
	}
	filePath = filepath.Join(rootPath, filePath, time.Now().Format(timeFormat))
	if err := MakeMultiDir(filePath); err != nil {
		return "", err
	}
	return filePath, nil
}

// GetFileNamesByDirPath 获取当前文件夹下的所有文件和文件夹名称（包括子文件夹和文件）
func GetFileNamesByDirPath(root string) ([]map[string]interface{}, error) {
	paths := make([]map[string]interface{}, 0)
	dirs, err := GetAllDirs(root)
	if err != nil {
		return paths, err
	}
	// 获取每个文件夹的第一级文件列表
	for _, dir := range dirs {
		var pathItem map[string]interface{}
		fileNames := make([]string, 0)
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			continue
		}
		pathItem = map[string]interface{}{
			"path":  strings.Replace(dir, root, "", 1),
			"files": []string{},
		}
		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".go") {
				continue
			}
			if strings.HasSuffix(file.Name(), "_test.go") {
				continue
			}
			fileNames = append(fileNames, strings.Replace(file.Name(), ".go", "", -1))
		}
		pathItem["files"] = fileNames
		paths = append(paths, pathItem)
	}
	return paths, nil
}

// getFilesInDir 获取指定目录下的所有文件名称
func getFilesInDir(dir string) ([]string, error) {
	files := make([]string, 0)
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 判断是否为文件
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			files = append(files, strings.Replace(info.Name(), ".go", "", -1))
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

// GetAllDirs 获取指定文件夹中所有文件夹路径
func GetAllDirs(root string) ([]string, error) {
	var dirs []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})
	return dirs, err
}
