package handler

import (
	"net/http"

	"kkg-backend/internal/session"

	"github.com/gin-gonic/gin"
)

const (
	accessTokenTTL  = session.AccessTokenTTL
	refreshTokenTTL = session.RefreshTokenTTL
)

func (h *Handler) setAuthCookies(c *gin.Context, sharedUserID int64, role string) error {
	tokens, err := h.sessions.IssuePair(c.Request.Context(), uint64(sharedUserID), role)
	if err != nil {
		return err
	}
	secure := c.Request.TLS != nil
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", tokens.AccessToken, int(accessTokenTTL.Seconds()), "/", "", secure, true)
	c.SetCookie("refresh_token", tokens.RefreshToken, int(refreshTokenTTL.Seconds()), "/", "", secure, true)
	return nil
}

func (h *Handler) clearAuthCookies(c *gin.Context) {
	accessToken, _ := c.Cookie("access_token")
	refreshToken, _ := c.Cookie("refresh_token")
	_ = h.sessions.DeletePair(c.Request.Context(), accessToken, refreshToken)
	secure := c.Request.TLS != nil
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", "", -1, "/", "", secure, true)
	c.SetCookie("refresh_token", "", -1, "/", "", secure, true)
	c.SetCookie("yuoj_session", "", -1, "/", "", secure, true)
}
