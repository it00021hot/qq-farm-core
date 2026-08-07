package tenant

// CreateReq 创建租户
type CreateReq struct {
	Code          string `json:"code" validate:"required,max=64"`
	Name          string `json:"name" validate:"required,max=128"`
	MaxUsers      uint   `json:"maxUsers"`
	MaxAccounts   uint   `json:"maxAccounts"`
	ExpireAt      uint   `json:"expireAt"`
	ContactName   string `json:"contactName"`
	ContactPhone  string `json:"contactPhone"`
	Remark        string `json:"remark"`
	AdminAccount  string `json:"adminAccount"`
	AdminPassword string `json:"adminPassword"`
	AdminNickName string `json:"adminNickName"`
}

// UpdateReq 更新租户
type UpdateReq struct {
	ID           uint64 `json:"id" validate:"required"`
	Name         string `json:"name" validate:"required,max=128"`
	MaxUsers     uint   `json:"maxUsers"`
	MaxAccounts  uint   `json:"maxAccounts"`
	ExpireAt     uint   `json:"expireAt"`
	ContactName  string `json:"contactName"`
	ContactPhone string `json:"contactPhone"`
	Remark       string `json:"remark"`
	Status       uint8  `json:"status" validate:"required,oneof=1 2"`
}

// StatusReq 启停
type StatusReq struct {
	ID     uint64 `json:"id" validate:"required"`
	Status uint8  `json:"status" validate:"required,oneof=1 2"`
}

// ListReq 列表
type ListReq struct {
	Current int    `json:"current" query:"current"`
	Size    int    `json:"size" query:"size"`
	Keyword string `json:"keyword" query:"keyword"`
	Status  uint8  `json:"status" query:"status"`
}

// IDReq ...
type IDReq struct {
	ID uint64 `json:"id" query:"id" validate:"required"`
}

// BindTenantReq 平台用户绑定租户
type BindTenantReq struct {
	AdminID   uint64   `json:"adminId" validate:"required"`
	TenantIDs []uint64 `json:"tenantIds" validate:"required"`
}
