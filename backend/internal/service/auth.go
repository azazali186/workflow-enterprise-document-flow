package service

import (
	"context"
	"strings"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/crypto"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/jwt"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/outbox"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"gorm.io/gorm"
)

// RegisterInput carries account creation payload.
type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Phone    string
}

// LoginInput carries authentication payload.
type LoginInput struct {
	Email    string
	Password string
}

// AuthResult is returned by login/register/refresh. CSRF is the double-submit
// value bound to this token; the SPA keeps it in memory and echoes it in the
// X-CSRF-Token header.
type AuthResult struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
	CSRF      string      `json:"csrf,omitempty"`
	User      *model.User `json:"user"`
}

// AuthService is the authentication contract.
type AuthService interface {
	Register(ctx context.Context, actor Actor, in RegisterInput) (*AuthResult, error)
	Login(ctx context.Context, in LoginInput, ip, ua string) (*AuthResult, error)
	Logout(ctx context.Context, actor Actor, userID string) error
	Refresh(ctx context.Context, tokenStr, userID string) (*AuthResult, error)
	Me(ctx context.Context, userID string) (*model.User, error)
}

// authService implements AuthService.
type authService struct {
	db    *gorm.DB
	users *repository.Repo[model.User]
	roles *repository.Repo[model.Role]
	logs  *repository.Repo[model.LoginLog]
	cache *cache.Client
	audit *AuditService
	ttl   time.Duration
}

// NewAuthService wires the auth implementation.
func NewAuthService(db *gorm.DB, users *repository.Repo[model.User], roles *repository.Repo[model.Role],
	logs *repository.Repo[model.LoginLog], c *cache.Client, audit *AuditService, ttl time.Duration) AuthService {
	return &authService{db: db, users: users, roles: roles, logs: logs, cache: c, audit: audit, ttl: ttl}
}

func (s *authService) Register(ctx context.Context, actor Actor, in RegisterInput) (*AuthResult, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if !strings.Contains(email, "@") {
		return nil, apperror.BadRequest("email is invalid")
	}
	if len(in.Password) < 8 {
		return nil, apperror.BadRequest("password must be at least 8 characters")
	}
	if _, err := s.users.FirstWhere(ctx, "email", email); err == nil {
		return nil, apperror.Conflict("email already registered")
	}
	hash, err := crypto.HashPassword(in.Password)
	if err != nil {
		return nil, apperror.Internal("failed to hash password", err)
	}
	encPhone, _ := crypto.Encrypt(in.Phone)
	user := &model.User{
		Email:        email,
		PasswordHash: hash,
		Name:         strings.TrimSpace(in.Name),
		Phone:        encPhone,
		Status:       model.UserActive,
	}
	var role model.Role
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		if err := tx.Where("code = ?", constant.RoleUser).First(&role).Error; err != nil {
			return err
		}
		return tx.Create(&model.UserRole{UserID: user.ID, RoleID: role.ID}).Error
	})
	if err != nil {
		return nil, err
	}
	_ = outbox.Enqueue(ctx, s.db, outbox.Event{
		AggregateType: "user", AggregateID: user.ID, EventType: constant.EventUserRegistered,
	}, map[string]any{"user_id": user.ID, "email": user.Email})
	_ = s.audit.Record(ctx, actor, constant.ActionCreate, "user", user.ID, nil, user)

	// Reload with roles so the issued token carries an accurate role_ids claim.
	if loaded, err := s.users.GetByID(ctx, user.ID); err == nil {
		user = loaded
	}

	result, err := s.issueSession(ctx, user)
	if err != nil {
		return nil, err
	}
	_ = s.recordLogin(ctx, user, true, user.Email, "", "", "")
	return result, nil
}

func (s *authService) Login(ctx context.Context, in LoginInput, ip, ua string) (*AuthResult, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	user, err := s.users.FirstWhere(ctx, "email", email)
	if err != nil {
		_ = s.recordLogin(ctx, nil, false, email, "invalid credentials", ip, ua)
		return nil, apperror.Unauthorized("invalid credentials")
	}
	if !crypto.CheckPassword(user.PasswordHash, in.Password) {
		_ = s.recordLogin(ctx, user, false, user.Email, "invalid credentials", ip, ua)
		return nil, apperror.Unauthorized("invalid credentials")
	}
	if user.Status != model.UserActive {
		_ = s.recordLogin(ctx, user, false, user.Email, "account "+user.Status, ip, ua)
		return nil, apperror.Forbidden("account is " + user.Status)
	}
	result, err := s.issueSession(ctx, user)
	if err != nil {
		return nil, err
	}
	_ = s.recordLogin(ctx, user, true, user.Email, "", ip, ua)
	_ = s.db.WithContext(ctx).Model(user).Update("last_login_at", time.Now()).Error
	return result, nil
}

func (s *authService) Logout(ctx context.Context, actor Actor, userID string) error {
	if err := s.cache.Del(SSOKey(userID)); err != nil {
		return err
	}
	return s.audit.Action(ctx, actor, constant.ActionLogout, "session", userID)
}

func (s *authService) Refresh(ctx context.Context, tokenStr, userID string) (*AuthResult, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, apperror.Unauthorized("session invalid")
	}
	if ok, err := RenewIfNeeded(s.cache, userID, SSOValue(tokenStr, userID), s.ttl); err != nil || !ok {
		return nil, apperror.Unauthorized("session expired")
	}
	return s.issueSession(ctx, user)
}

func (s *authService) Me(ctx context.Context, userID string) (*model.User, error) {
	return s.users.GetByID(ctx, userID)
}

func (s *authService) issueSession(ctx context.Context, user *model.User) (*AuthResult, error) {
	roleIDs := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roleIDs = append(roleIDs, r.ID)
	}
	token, exp, err := jwt.Generate(user.ID, user.Email, roleIDs)
	if err != nil {
		return nil, apperror.Internal("failed to sign token", err)
	}
	if err := s.cache.Set(SSOKey(user.ID), SSOValue(token, user.ID), s.ttl); err != nil {
		return nil, apperror.Internal("failed to persist session", err)
	}
	return &AuthResult{Token: token, ExpiresAt: exp, CSRF: jwt.CSRFFor(token), User: user}, nil
}

func (s *authService) recordLogin(ctx context.Context, user *model.User, ok bool, email, reason, ip, ua string) error {
	entry := &model.LoginLog{
		UserID:        nil,
		Email:         email,
		Status:        "success",
		FailureReason: "",
		IPAddress:     ip,
		UserAgent:     ua,
	}
	if !ok {
		entry.Status = "failure"
		entry.FailureReason = reason
	}
	if user != nil {
		entry.UserID = model.NullableString(user.ID)
	}
	if entry.Email == "" {
		entry.Email = "-"
	}
	return s.logs.Create(ctx, entry)
}

var _ AuthService = (*authService)(nil)
