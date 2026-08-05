package auth

type LoginReq struct {
	Account  string `form:"userName" json:"userName" validate:"required"`
	Password string `form:"password" json:"password" validate:"required"`
}

type RefreshReq struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

// ChangePasswordReq 登录用户修改密码
type ChangePasswordReq struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=6,max=32"`
}

type LoginTokenResp struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	TenantID     uint64 `json:"tenantId"`
}

// ElegantMeta soybean 动态路由 meta
type ElegantMeta struct {
	Title      string `json:"title"`
	Icon       string `json:"icon,omitempty"`
	Order      int    `json:"order,omitempty"`
	HideInMenu bool   `json:"hideInMenu,omitempty"`
}

// MenuNode 前端菜单树节点（目录/菜单）
type MenuNode struct {
	ID           uint64     `json:"id"`
	Name         string     `json:"name"`
	Alias        string     `json:"alias"`
	FURL         string     `json:"fUrl"`
	Icon         string     `json:"icon"`
	ParentID     uint64     `json:"parentId"`
	ResourceType uint8      `json:"resourceType"`
	HideInMenu   uint8      `json:"hideInMenu"`
	SortOrder    uint16     `json:"sortOrder"`
	Children     []MenuNode `json:"children,omitempty"`
}

// InfoResp 当前用户信息与权限（对齐 soybean UserInfo + 扩展）
type InfoResp struct {
	UserID   string     `json:"userId"`
	UserName string     `json:"userName"`
	NickName string     `json:"nickName"`
	TenantID uint64     `json:"tenantId"`
	UUID     string     `json:"uuid"`
	RoleIDs  []string   `json:"roleIds"`
	Roles    []string   `json:"roles"`
	Menus    []MenuNode `json:"menus"`
	Buttons  []string   `json:"buttons"`
}

// ElegantRoute soybean ElegantConstRoute 子集
type ElegantRoute struct {
	Name      string         `json:"name"`
	Path      string         `json:"path"`
	Component string         `json:"component,omitempty"`
	Meta      ElegantMeta    `json:"meta"`
	Children  []ElegantRoute `json:"children,omitempty"`
}

// UserRouteResp GET /auth/user-routes
type UserRouteResp struct {
	Routes []ElegantRoute `json:"routes"`
	Home   string         `json:"home"`
}
