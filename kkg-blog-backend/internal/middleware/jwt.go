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
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.Unauthorized(c, "missing or invalid authorization header")
			c.Abort()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
		claims, err := security.ParseJWT(token, secret)
		if err != nil {
			response.Unauthorized(c, "invalid token")
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
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
		claims, err := security.ParseJWT(token, secret)
		if err == nil {
			c.Set(ctxUserID, claims.UserID)
			c.Set(ctxRole, claims.Role)
		}
		c.Next()
	}
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
