package role

type CreateReq struct {
	ParentID uint64 `json:"parent_id"`
	Name     string `json:"name" validate:"required,max=64"`
	Code     string `json:"code" validate:"required,max=32"`
	Desc     string `json:"desc"`
	RoleType uint8  `json:"role_type" validate:"required,oneof=1 2"`
	Status   uint8  `json:"status" validate:"required,oneof=1 2"`
}

type UpdateReq struct {
	ID       uint64 `json:"id" validate:"required"`
	ParentID uint64 `json:"parent_id"`
	Name     string `json:"name" validate:"required,max=64"`
	Code     string `json:"code" validate:"required,max=32"`
	Desc     string `json:"desc"`
	RoleType uint8  `json:"role_type" validate:"required,oneof=1 2"`
	Status   uint8  `json:"status" validate:"required,oneof=1 2"`
}

type IDReq struct {
	ID uint64 `json:"id" query:"id" validate:"required"`
}

type AuthReq struct {
	RoleID      uint64 `json:"role_id" validate:"required"`
	ResourceIDs string `json:"resource_ids"` // 逗号分隔
}
