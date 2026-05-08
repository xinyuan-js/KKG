package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"awesomeProject/internal/model"
	"awesomeProject/internal/repository"
	"awesomeProject/pkg/security"
)

type AuthService struct {
	users     *repository.UserRepository
	jwtSecret string
}

func NewAuthService(users *repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret}
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

func (s *AuthService) Login(account string, password string) (string, *model.User, error) {
	user, err := s.users.GetByUsernameOrEmail(strings.TrimSpace(account))
	if err != nil {
		return "", nil, err
	}
	if user == nil {
		return "", nil, errors.New("invalid credentials")
	}
	if user.Status != 1 {
		if user.Status == 0 {
			return "", nil, errors.New("账号已被禁用，请联系管理员")
		}
		return "", nil, errors.New("账号不存在或已被隐藏，请联系管理员")
	}

	if !security.CheckPassword(user.PasswordHash, password) {
		return "", nil, errors.New("invalid credentials")
	}

	token, err := security.GenerateJWT(user.ID, user.Role, s.jwtSecret, 24*time.Hour)
	if err != nil {
		return "", nil, fmt.Errorf("generate jwt failed: %w", err)
	}

	return token, user, nil
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
