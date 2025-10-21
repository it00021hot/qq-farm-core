package excel

import (
	"io"

	"github.com/xuri/excelize/v2"
)

// ReadFile 读取excel文件
func ReadFile(filePath, sheet string) ([]map[string]interface{}, error) {
	var (
		firstCell   = make([]string, 0)
		productData = make([]map[string]interface{}, 0)
	)
	// 获取excel数据
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()
	if sheet == "" {
		sheet = "Sheet1"
	}
	// Get all the rows in the Sheet1.
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	for key, row := range rows {
		if key == 0 {
			firstCell = row
			continue
		}
		item := make(map[string]interface{}, 0)
		for i, s := range firstCell {
			item[s] = row[i]
		}
		productData = append(productData, item)
	}
	return productData, nil
}

// ReadStream 读取excel文件流
func ReadStream(r io.Reader, sheet string) ([]map[string]interface{}, error) {
	var (
		firstCell   = make([]string, 0)
		productData = make([]map[string]interface{}, 0)
	)
	// 获取excel数据
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()
	if sheet == "" {
		sheet = "Sheet1"
	}
	// Get all the rows in the Sheet1.
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	for key, row := range rows {
		if key == 0 {
			firstCell = row
			continue
		}
		item := make(map[string]interface{}, 0)
		for i, s := range firstCell {
			item[s] = ""
			if i < len(row) {
				item[s] = row[i]
			}
		}
		productData = append(productData, item)
	}
	return productData, nil
}
