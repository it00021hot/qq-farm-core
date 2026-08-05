package attachment

import "github.com/MQEnergy/go-skeleton/pkg/upload"

type UploadReq struct {
	upload.File
	FilePath string `form:"filePath" json:"filePath"`
}

type TplDownloadReq struct {
	ID uint64 `query:"id" json:"id" validate:"required"`
}

type TplImportReq struct {
	AttachmentID uint64 `json:"attachmentId" validate:"required"`
	SheetName    string `json:"sheetName"`
}

type ReadFileReq struct {
	FilePath    string `uri:"file_path" query:"filePath" json:"filePath" validate:"required"`
	XOssProcess string `uri:"x-oss-process" query:"xOssProcess" json:"xOssProcess"`
}

type AccessURLReq struct {
	FilePath string `json:"filePath" form:"filePath" validate:"required"`
}

type AccessURLResp struct {
	FilePath  string `json:"filePath"`
	SignedURL string `json:"signedUrl"`
	Expire    int64  `json:"expire"`
}

type ListReq struct {
	Current    int    `json:"current" query:"current"`
	Size       int    `json:"size" query:"size"`
	Keyword    string `json:"keyword" query:"keyword"`
	AttachType uint8  `json:"attachType" query:"attachType"`
	Status     uint8  `json:"status" query:"status"`
}

type IDReq struct {
	ID uint64 `json:"id" query:"id" validate:"required"`
}

type StatusReq struct {
	ID     uint64 `json:"id" validate:"required"`
	Status uint8  `json:"status" validate:"required,oneof=0 1"`
}

type DeleteReq struct {
	ID uint64 `json:"id" validate:"required"`
}
