package admin

type CreateReq struct {
	Account  string `json:"account" validate:"required,max=64"`
	Password string `json:"password" validate:"required,min=6"`
	NickName string `json:"nickName" validate:"required,max=64"`
	RealName string `json:"realName"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	RoleIDs  string `json:"roleIds" validate:"required"`
	Status   uint8  `json:"status" validate:"required,oneof=1 2"`
}

type UpdateReq struct {
	ID       uint64 `json:"id" validate:"required"`
	NickName string `json:"nickName" validate:"required,max=64"`
	RealName string `json:"realName"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	RoleIDs  string `json:"roleIds" validate:"required"`
	Status   uint8  `json:"status" validate:"required,oneof=1 2"`
	Password string `json:"password"`
}

type ListReq struct {
	Current int    `json:"current" query:"current"`
	Size    int    `json:"size" query:"size"`
	Keyword string `json:"keyword" query:"keyword"`
	Status  uint8  `json:"status" query:"status"`
}

type IDReq struct {
	ID uint64 `json:"id" query:"id" validate:"required"`
}

type StatusReq struct {
	ID     uint64 `json:"id" validate:"required"`
	Status uint8  `json:"status" validate:"required,oneof=1 2"`
}

// PlatformCreateReq 创建平台用户（tenant_id=0）
type PlatformCreateReq struct {
	Account   string   `json:"account" validate:"required,max=64"`
	Password  string   `json:"password" validate:"required,min=6"`
	NickName  string   `json:"nickName" validate:"required,max=64"`
	RoleIDs   string   `json:"roleIds" validate:"required"`
	TenantIDs []uint64 `json:"tenantIds"`
	Status    uint8    `json:"status" validate:"required,oneof=1 2"`
}
