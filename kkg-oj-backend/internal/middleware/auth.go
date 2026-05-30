package middleware

import (
	"errors"
	"strings"
	"yuoj-go-backend/internal/common"
	"yuoj-go-backend/internal/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type blogClaims struct {
	UserID    float64 `json:"user_id"`
	TokenType string  `json:"token_type"`
	jwt.RegisteredClaims
}

func MustLogin(userSvc *service.UserService, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := accessTokenFromRequest(c)
		if token != "" && userSvc != nil && jwtSecret != "" {
			sharedID, err := parseBlogUserID(token, jwtSecret)
			if err == nil && sharedID > 0 {
				u, ensureErr := userSvc.EnsureFromSharedUserID(sharedID)
				if ensureErr == nil && u != nil {
					c.Set("loginUserId", u.ID)
					c.Next()
					return
				}
			}
		}

		sess := sessions.Default(c)
		userID := sess.Get(common.UserLoginState)
		if userID != nil {
			c.Set("loginUserId", userID)
			c.Next()
			return
		}

		panic(common.NewBizError(common.NotLoginError, "未登录"))
	}
}

func parseBlogUserID(tokenString, secret string) (int64, error) {
	token, err := jwt.ParseWithClaims(tokenString, &blogClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return 0, err
	}
	claims, ok := token.Claims.(*blogClaims)
	if !ok || !token.Valid {
		return 0, errors.New("invalid token")
	}
	if claims.TokenType != "" && claims.TokenType != "access" {
		return 0, errors.New("invalid token type")
	}
	if claims.UserID <= 0 {
		return 0, errors.New("invalid user id")
	}
	return int64(claims.UserID), nil
}

func accessTokenFromRequest(c *gin.Context) string {
	if token, err := c.Cookie("access_token"); err == nil && strings.TrimSpace(token) != "" {
		return strings.TrimSpace(token)
	}
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	return ""
}
