package attachment

import "github.com/MQEnergy/go-skeleton/pkg/upload"

type UploadReq struct {
	upload.File
	FilePath string `form:"file_path"`
}

type TplDownloadReq struct {
	ID uint64 `query:"id" json:"id" validate:"required"`
}

type TplImportReq struct {
	AttachmentID uint64 `json:"attachment_id" validate:"required"` // 附件ID
	SheetName    string `json:"sheet_name"`                        // excel文件的sheet名称
}

type ReadFileReq struct {
	FilePath    string `uri:"file_path" query:"file_path" validate:"required"` // 文件路径
	XOssProcess string `uri:"x-oss-process" query:"x-oss-process"`             // x-oss-process 图片适用 示例：image/resize,m_lfit,h_200,w_200 image/resize,m_fixed,w_200,h_100
}

// AccessURLReq 置换私有对象临时访问地址
type AccessURLReq struct {
	FilePath string `json:"file_path" form:"file_path" validate:"required"` // 附件 attach_url / file_path
}

// AccessURLResp 临时访问地址响应
type AccessURLResp struct {
	FilePath  string `json:"file_path"`
	SignedURL string `json:"signed_url"`
	Expire    int64  `json:"expire"`
}
