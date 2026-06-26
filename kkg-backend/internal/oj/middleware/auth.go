package middleware

import (
	"kkg-backend/internal/oj/common"
	"kkg-backend/internal/oj/service"
	"kkg-backend/internal/session"
	"strings"

	"github.com/gin-gonic/gin"
)

func MustLogin(userSvc *service.UserService, sessions *session.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := accessTokenFromRequest(c)
		if token != "" && userSvc != nil && sessions != nil {
			data, err := sessions.ValidateAccess(c.Request.Context(), token)
			if err == nil && data.UserID > 0 {
				sharedID := int64(data.UserID)
				u, ensureErr := userSvc.EnsureFromSharedUserID(sharedID)
				if ensureErr == nil && u != nil {
					c.Set("loginUserId", u.ID)
					c.Next()
					return
				}
			}
		}

		panic(common.NewBizError(common.NotLoginError, "未登录"))
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
