package menu

type CreateReq struct {
	Name         string `json:"name" validate:"required"`
	Alias        string `json:"alias" validate:"required"`
	Desc         string `json:"desc"`
	FURL         string `json:"fUrl"`
	BURL         string `json:"bUrl"`
	Methods      string `json:"methods"`
	ParentID     uint64 `json:"parentId"`
	Path         string `json:"path"`
	ResourceType uint8  `json:"resourceType" validate:"required,oneof=1 2 3"`
	Icon         string `json:"icon"`
	HideInMenu   uint8  `json:"hideInMenu" validate:"required,oneof=1 2"`
	Status       uint8  `json:"status" validate:"required,oneof=1 2"`
	SortOrder    uint16 `json:"sortOrder"`
}

type UpdateReq struct {
	ID           uint64 `json:"id" validate:"required"`
	Name         string `json:"name" validate:"required"`
	Alias        string `json:"alias" validate:"required"`
	Desc         string `json:"desc"`
	FURL         string `json:"fUrl"`
	BURL         string `json:"bUrl"`
	Methods      string `json:"methods"`
	ParentID     uint64 `json:"parentId"`
	Path         string `json:"path"`
	ResourceType uint8  `json:"resourceType" validate:"required,oneof=1 2 3"`
	Icon         string `json:"icon"`
	HideInMenu   uint8  `json:"hideInMenu" validate:"required,oneof=1 2"`
	Status       uint8  `json:"status" validate:"required,oneof=1 2"`
	SortOrder    uint16 `json:"sortOrder"`
}

type IDReq struct {
	ID uint64 `json:"id" query:"id" validate:"required"`
}
