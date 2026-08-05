package jwtauth

import (
	"errors"
	"fmt"
	"time"

	"github.com/MQEnergy/go-skeleton/pkg/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type JWT struct {
	claims jwt.MapClaims
	token  *jwt.Token
	cfg    *config.Config
}

func New(cfg *config.Config) *JWT {
	return &JWT{
		claims: jwt.MapClaims{
			"iss": cfg.GetString("jwt.issuer"),
		},
		cfg: cfg,
	}
}

func (j *JWT) accessExpire() time.Duration {
	sec := j.cfg.GetInt64("jwt.accessExpire")
	if sec <= 0 {
		sec = j.cfg.GetInt64("jwt.expire")
	}
	if sec <= 0 {
		sec = 7200
	}
	return time.Duration(sec) * time.Second
}

func (j *JWT) refreshExpire() time.Duration {
	sec := j.cfg.GetInt64("jwt.refreshExpire")
	if sec <= 0 {
		sec = 864000
	}
	return time.Duration(sec) * time.Second
}

func (j *JWT) WithClaims(sub jwt.MapClaims) *JWT {
	j.claims["sub"] = sub
	j.token = jwt.NewWithClaims(jwt.SigningMethodHS256, j.claims)
	return j
}

func (j *JWT) GenerateToken() (string, error) {
	if j.token == nil {
		return "", errors.New("claims not set")
	}
	j.claims["exp"] = jwt.NewNumericDate(time.Now().Add(j.accessExpire()))
	j.claims["typ"] = TokenTypeAccess
	j.claims["jti"] = uuid.NewString()
	j.token = jwt.NewWithClaims(jwt.SigningMethodHS256, j.claims)
	return j.token.SignedString([]byte(j.cfg.GetString("jwt.secret")))
}

// GenerateTokenPair 签发 access + refresh
func (j *JWT) GenerateTokenPair(sub jwt.MapClaims) (access string, refresh string, err error) {
	now := time.Now()
	accessJTI := uuid.NewString()
	refreshJTI := uuid.NewString()

	accessClaims := jwt.MapClaims{
		"iss": j.cfg.GetString("jwt.issuer"),
		"exp": jwt.NewNumericDate(now.Add(j.accessExpire())),
		"iat": jwt.NewNumericDate(now),
		"typ": TokenTypeAccess,
		"jti": accessJTI,
		"sub": sub,
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	access, err = accessToken.SignedString([]byte(j.cfg.GetString("jwt.secret")))
	if err != nil {
		return "", "", err
	}

	refreshClaims := jwt.MapClaims{
		"iss": j.cfg.GetString("jwt.issuer"),
		"exp": jwt.NewNumericDate(now.Add(j.refreshExpire())),
		"iat": jwt.NewNumericDate(now),
		"typ": TokenTypeRefresh,
		"jti": refreshJTI,
		"sub": sub,
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refresh, err = refreshToken.SignedString([]byte(j.cfg.GetString("jwt.secret")))
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (j *JWT) RefreshTTL() time.Duration {
	return j.refreshExpire()
}

// ParseToken parse token
func (j *JWT) ParseToken(token string) (jwt.MapClaims, error) {
	_token, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(j.cfg.GetString("jwt.secret")), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := _token.Claims.(jwt.MapClaims); ok && _token.Valid {
		return claims, nil
	}
	return nil, errors.New("jwt 解析验证后失败")
}
