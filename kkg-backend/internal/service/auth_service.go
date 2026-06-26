package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"kkg-backend/internal/model"
	"kkg-backend/internal/repository"
	"kkg-backend/internal/session"
	"kkg-backend/pkg/security"
)

type AuthService struct {
	users    *repository.UserRepository
	sessions *session.Manager
}

type TokenPair = session.Pair

const (
	AccessTokenTTL  = session.AccessTokenTTL
	RefreshTokenTTL = session.RefreshTokenTTL

	legacyOJPasswordPrefix = "$legacy_oj_md5$"
	legacyOJPasswordSalt   = "yupi"
)

func NewAuthService(users *repository.UserRepository, sessions *session.Manager) *AuthService {
	return &AuthService{users: users, sessions: sessions}
}

func (s *AuthService) Register(username string, email string, password string) (*model.User, error) {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	if username == "" || email == "" || password == "" {
		return nil, errors.New("username email and password are required")
	}
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	exists, err := s.users.ExistsByUsernameOrEmail(username, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("username or email already exists")
	}

	hash, err := security.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password failed: %w", err)
	}

	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Role:         "user",
		Status:       1,
	}
	if err := s.users.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, account string, password string) (*TokenPair, *model.User, error) {
	user, err := s.users.GetByUsernameOrEmail(strings.TrimSpace(account))
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, errors.New("invalid credentials")
	}
	if user.Status != 1 {
		if user.Status == 0 {
			return nil, nil, errors.New("账号已被禁用，请联系管理员")
		}
		return nil, nil, errors.New("账号不存在或已被隐藏，请联系管理员")
	}

	if !security.CheckPassword(user.PasswordHash, password) {
		ok, upgradeErr := s.checkAndUpgradeLegacyOJPassword(user, password)
		if upgradeErr != nil {
			return nil, nil, upgradeErr
		}
		if !ok {
			return nil, nil, errors.New("invalid credentials")
		}
	}

	tokens, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	return tokens, user, nil
}

func (s *AuthService) checkAndUpgradeLegacyOJPassword(user *model.User, password string) (bool, error) {
	legacyHash, ok := strings.CutPrefix(user.PasswordHash, legacyOJPasswordPrefix)
	if !ok {
		return false, nil
	}

	sum := md5.Sum([]byte(legacyOJPasswordSalt + password))
	if !strings.EqualFold(hex.EncodeToString(sum[:]), strings.TrimSpace(legacyHash)) {
		return false, nil
	}

	hash, err := security.HashPassword(password)
	if err != nil {
		return false, fmt.Errorf("hash password failed: %w", err)
	}
	user.PasswordHash = hash
	if err := s.users.Update(user); err != nil {
		return false, err
	}
	return true, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, *model.User, error) {
	tokens, data, err := s.sessions.Refresh(ctx, strings.TrimSpace(refreshToken))
	if err != nil {
		return nil, nil, errors.New("invalid refresh token")
	}
	user, err := s.users.GetByID(data.UserID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil || user.Status != 1 {
		return nil, nil, errors.New("invalid refresh token user")
	}
	return tokens, user, nil
}

func (s *AuthService) Logout(ctx context.Context, accessToken string, refreshToken string) error {
	return s.sessions.DeletePair(ctx, strings.TrimSpace(accessToken), strings.TrimSpace(refreshToken))
}

func (s *AuthService) issueTokenPair(ctx context.Context, user *model.User) (*TokenPair, error) {
	tokens, err := s.sessions.IssuePair(ctx, user.ID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate session token failed: %w", err)
	}
	return tokens, nil
}

func (s *AuthService) GetProfile(userID uint64) (*model.User, error) {
	if userID == 0 {
		return nil, errors.New("invalid user")
	}
	user, err := s.users.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (s *AuthService) UpdateProfile(userID uint64, username string, email string, avatarURL *string) (*model.User, error) {
	if userID == 0 {
		return nil, errors.New("invalid user")
	}
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	if username == "" || email == "" {
		return nil, errors.New("username and email are required")
	}

	user, err := s.users.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	exists, err := s.users.ExistsByUsernameOrEmailExcludeID(username, email, userID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("username or email already exists")
	}

	user.Username = username
	user.Email = email
	if avatarURL != nil {
		user.AvatarURL = strings.TrimSpace(*avatarURL)
	}
	if err := s.users.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) ChangePassword(userID uint64, oldPassword string, newPassword string) error {
	if userID == 0 {
		return errors.New("invalid user")
	}
	oldPassword = strings.TrimSpace(oldPassword)
	newPassword = strings.TrimSpace(newPassword)
	if oldPassword == "" || newPassword == "" {
		return errors.New("old_password and new_password are required")
	}
	if len(newPassword) < 8 {
		return errors.New("new password must be at least 8 characters")
	}

	user, err := s.users.GetByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	if !security.CheckPassword(user.PasswordHash, oldPassword) {
		return errors.New("old password is incorrect")
	}

	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password failed: %w", err)
	}
	user.PasswordHash = hash
	return s.users.Update(user)
}
