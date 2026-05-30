package middleware

import (
	"strings"

	"awesomeProject/internal/model"
	"awesomeProject/pkg/response"
	"awesomeProject/pkg/security"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	ctxUserID = "user_id"
	ctxRole   = "role"
)

const (
	RoleUser       = "user"
	RoleAdmin      = "admin"
	RoleSuperAdmin = "super_admin"
)

func JWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := accessTokenFromRequest(c)
		if token == "" {
			response.Unauthorized(c, "missing access token")
			c.Abort()
			return
		}

		claims, err := security.ParseJWT(token, secret)
		if err != nil {
			response.Unauthorized(c, "invalid token")
			c.Abort()
			return
		}
		if claims.TokenType != "" && claims.TokenType != "access" {
			response.Unauthorized(c, "invalid token type")
			c.Abort()
			return
		}

		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxRole, claims.Role)
		c.Next()
	}
}

func MustUserID(c *gin.Context) uint64 {
	v, ok := c.Get(ctxUserID)
	if !ok {
		return 0
	}
	id, ok := v.(uint64)
	if !ok {
		return 0
	}
	return id
}

func MustRole(c *gin.Context) string {
	v, ok := c.Get(ctxRole)
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func IsAdminRole(role string) bool {
	return role == RoleAdmin || role == RoleSuperAdmin
}

func OptionalJWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := accessTokenFromRequest(c)
		if token == "" {
			c.Next()
			return
		}
		claims, err := security.ParseJWT(token, secret)
		if err == nil && (claims.TokenType == "" || claims.TokenType == "access") {
			c.Set(ctxUserID, claims.UserID)
			c.Set(ctxRole, claims.Role)
		}
		c.Next()
	}
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

func RequireActiveUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := MustUserID(c)
		if userID == 0 {
			response.Unauthorized(c, "invalid user context")
			c.Abort()
			return
		}
		var user model.User
		if err := db.Select("id, role, status").First(&user, userID).Error; err != nil {
			response.Unauthorized(c, "invalid token user")
			c.Abort()
			return
		}
		if user.Status != 1 {
			if user.Status == 0 {
				response.Forbidden(c, "账号已被禁用，请联系管理员")
			} else {
				response.Forbidden(c, "账号不存在或已被隐藏，请联系管理员")
			}
			c.Abort()
			return
		}
		c.Set(ctxRole, user.Role)
		c.Next()
	}
}
