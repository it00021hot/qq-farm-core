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
	FilePath    string `params:"file_path" query:"file_path" validate:"required"` // 文件路径
	XOssProcess string `params:"x-oss-process" query:"x-oss-process"`             // x-oss-process 图片适用 示例：image/resize,m_lfit,h_200,w_200 image/resize,m_fixed,w_200,h_100
}
