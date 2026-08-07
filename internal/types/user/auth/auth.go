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
}

// InfoResp 当前用户信息与权限（登录即全权限）
type InfoResp struct {
	UserID   string   `json:"userId"`
	UserName string   `json:"userName"`
	NickName string   `json:"nickName"`
	UUID     string   `json:"uuid"`
	RoleIDs  []string `json:"roleIds"`
	Roles    []string `json:"roles"`
	Menus    any      `json:"menus"`
	Buttons  []string `json:"buttons"`
}
