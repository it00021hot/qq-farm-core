package auth

// LoginReq ...
type LoginReq struct {
	Account  string `form:"account" json:"account" validate:"required"`
	Password string `form:"password" json:"password" validate:"required"`
}
