package handler

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"

	"awesomeProject/internal/model"
	"awesomeProject/internal/repository"
	"awesomeProject/internal/service"
	"awesomeProject/pkg/response"
	"awesomeProject/pkg/security"

	"github.com/gin-gonic/gin"
)

type InternalHandler struct {
	users *repository.UserRepository
	posts *service.PostService
}

func NewInternalHandler(users *repository.UserRepository, posts *service.PostService) *InternalHandler {
	return &InternalHandler{users: users, posts: posts}
}

type internalAgentPostReq struct {
	QuestionID   uint64   `json:"question_id"`
	Title        string   `json:"title" binding:"required,max=255"`
	Summary      string   `json:"summary" binding:"max=512"`
	Markdown     string   `json:"markdown" binding:"required"`
	Tags         []string `json:"tags" binding:"omitempty,dive,max=32"`
	Account      string   `json:"account" binding:"required,max=64"`
	Email        string   `json:"email" binding:"required,email,max=128"`
	DisplayName  string   `json:"display_name" binding:"max=64"`
	SourceSystem string   `json:"source_system" binding:"max=64"`
}

func (h *InternalHandler) PublishAgentPost(c *gin.Context) {
	var req internalAgentPostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	user, err := h.ensureAgentUser(req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	tags := req.Tags
	if len(tags) == 0 {
		tags = []string{"题解", "agent", "oj"}
		if req.QuestionID > 0 {
			tags = append(tags, "q"+strconv.FormatUint(req.QuestionID, 10))
		}
	}
	post, err := h.posts.CreateDraft(user.ID, req.Title, "", req.Summary, tags, req.Markdown, nil)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	published, err := h.posts.PublishDraft(post.ID, user.ID, 1)
	if err != nil {
		h.handleInternalPostErr(c, err)
		return
	}
	response.OK(c, gin.H{
		"id":  published.ID,
		"url": "/posts/" + strconv.FormatUint(published.ID, 10),
	})
}

func (h *InternalHandler) ensureAgentUser(req internalAgentPostReq) (*model.User, error) {
	account := strings.TrimSpace(req.Account)
	email := strings.TrimSpace(req.Email)
	user, err := h.users.GetByUsernameOrEmail(account)
	if err != nil {
		return nil, err
	}
	if user == nil {
		user, err = h.users.GetByUsernameOrEmail(email)
		if err != nil {
			return nil, err
		}
	}
	if user != nil {
		updates := false
		if user.Status != 1 {
			user.Status = 1
			updates = true
		}
		if strings.TrimSpace(user.Role) == "" {
			user.Role = "user"
			updates = true
		}
		if updates {
			if err := h.users.Update(user); err != nil {
				return nil, err
			}
		}
		return user, nil
	}

	passwordHash, err := security.HashPassword(randomSecret())
	if err != nil {
		return nil, err
	}
	user = &model.User{
		Username:     account,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         "user",
		Status:       1,
	}
	if err := h.users.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (h *InternalHandler) handleInternalPostErr(c *gin.Context, err error) {
	if err == nil {
		return
	}
	response.BadRequest(c, err.Error())
}

func randomSecret() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "internal-agent-user-password"
	}
	return hex.EncodeToString(b[:])
}
