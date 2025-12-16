package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type CustomClaims struct {
	ID            int64    `json:"id,omitempty"`
	UserID        string   `json:"user_id,omitempty"`
	Email         string   `json:"email,omitempty"`
	Roles         []string `json:"roles,omitempty"`
	NickName      string   `json:"nickname,omitempty"`
	AuthorityTime uint64   `json:"authority_time,omitempty"` // TODO: 用户权限到期时间(占位)
	jwt.RegisteredClaims
}

// CreateToken generate token
func CreateToken(c CustomClaims, key string) (string, error) {
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signedString, err := claims.SignedString([]byte(key))
	if err != nil {
		return "", errors.New("generate token failed" + err.Error())
	}
	return signedString, nil
}

func ParseToken(tokenString, key string) (*CustomClaims, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method %v", ErrInvalidToken, t.Header["alg"])
		}
		return []byte(key), nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidToken, err.Error())
	}
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
