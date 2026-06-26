package handler

import (
	"kkg-backend/internal/middleware"
	"kkg-backend/internal/service"
	"kkg-backend/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerReq struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Email    string `json:"email" binding:"required,email,max=128"`
	Password string `json:"password" binding:"required,min=8,max=64"`
}

type loginReq struct {
	Account  string `json:"account" binding:"required,max=128"`
	Password string `json:"password" binding:"required,min=8,max=64"`
}

type updateProfileReq struct {
	Username  string  `json:"username" binding:"required,min=3,max=64"`
	Email     string  `json:"email" binding:"required,email,max=128"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty,max=512"`
}

type changePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required,min=8,max=64"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=64"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, err := h.auth.Register(req.Username, req.Email, req.Password)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.OK(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"avatar_url": user.AvatarURL,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tokens, user, err := h.auth.Login(c.Request.Context(), req.Account, req.Password)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	setAuthCookies(c, tokens)

	response.OK(c, gin.H{
		"access_token": tokens.AccessToken,
		"user": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"avatar_url": user.AvatarURL,
			"role":       user.Role,
		},
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		response.Unauthorized(c, "missing refresh token")
		return
	}
	tokens, user, err := h.auth.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		clearAuthCookies(c)
		response.Unauthorized(c, err.Error())
		return
	}
	setAuthCookies(c, tokens)
	response.OK(c, gin.H{
		"user": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"avatar_url": user.AvatarURL,
			"role":       user.Role,
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	accessToken, _ := c.Cookie("access_token")
	refreshToken, _ := c.Cookie("refresh_token")
	_ = h.auth.Logout(c.Request.Context(), accessToken, refreshToken)
	clearAuthCookies(c)
	response.OK(c, gin.H{"logged_out": true})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	user, err := h.auth.GetProfile(userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"avatar_url": user.AvatarURL,
		"role":       user.Role,
	})
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	var req updateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	user, err := h.auth.UpdateProfile(userID, req.Username, req.Email, req.AvatarURL)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"avatar_url": user.AvatarURL,
		"role":       user.Role,
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.auth.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"changed": true})
}

func setAuthCookies(c *gin.Context, tokens *service.TokenPair) {
	c.SetSameSite(http.SameSiteLaxMode)
	secure := c.Request.TLS != nil
	c.SetCookie("access_token", tokens.AccessToken, int(service.AccessTokenTTL.Seconds()), "/", "", secure, true)
	c.SetCookie("refresh_token", tokens.RefreshToken, int(service.RefreshTokenTTL.Seconds()), "/", "", secure, true)
}

func clearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	secure := c.Request.TLS != nil
	c.SetCookie("access_token", "", -1, "/", "", secure, true)
	c.SetCookie("refresh_token", "", -1, "/", "", secure, true)
}
