package menu

type CreateReq struct {
	Name         string `json:"name" validate:"required"`
	Alias        string `json:"alias" validate:"required"`
	Desc         string `json:"desc"`
	FURL         string `json:"f_url"`
	BURL         string `json:"b_url"`
	ParentID     uint64 `json:"parent_id"`
	Path         string `json:"path"`
	ResourceType uint8  `json:"resource_type" validate:"required,oneof=1 2 3"`
	Icon         string `json:"icon"`
	Status       uint8  `json:"status" validate:"required,oneof=1 2"`
	SortOrder    uint16 `json:"sort_order"`
}

type UpdateReq struct {
	ID           uint64 `json:"id" validate:"required"`
	Name         string `json:"name" validate:"required"`
	Alias        string `json:"alias" validate:"required"`
	Desc         string `json:"desc"`
	FURL         string `json:"f_url"`
	BURL         string `json:"b_url"`
	ParentID     uint64 `json:"parent_id"`
	Path         string `json:"path"`
	ResourceType uint8  `json:"resource_type" validate:"required,oneof=1 2 3"`
	Icon         string `json:"icon"`
	Status       uint8  `json:"status" validate:"required,oneof=1 2"`
	SortOrder    uint16 `json:"sort_order"`
}

type IDReq struct {
	ID uint64 `json:"id" query:"id" validate:"required"`
}
