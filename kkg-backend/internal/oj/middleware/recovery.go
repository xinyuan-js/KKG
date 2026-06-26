package middleware

import (
	"net/http"
	"kkg-backend/internal/oj/common"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				switch e := rec.(type) {
				case *common.BizError:
					c.JSON(http.StatusOK, common.Error(e.Code, e.Message))
				case error:
					c.JSON(http.StatusOK, common.Error(common.SystemError, e.Error()))
				default:
					c.JSON(http.StatusOK, common.Error(common.SystemError, "系统错误"))
				}
				c.Abort()
			}
		}()
		c.Next()
	}
}
