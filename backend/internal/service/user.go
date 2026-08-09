package service

import (
	"context"
	"strings"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/crypto"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/pagination"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"gorm.io/gorm"
)

// CreateUserInput carries admin-created account payload.
type CreateUserInput struct {
	Email    string
	Password string
	Name     string
	Phone    string
	RoleIDs  []string
}

// UpdateUserInput carries admin account updates.
type UpdateUserInput struct {
	ID      string
	Name    string
	Phone   string
	Status  string
	RoleIDs []string
}

// UserService is the user administration contract.
type UserService interface {
	Create(ctx context.Context, actor Actor, in CreateUserInput) (*model.User, error)
	Update(ctx context.Context, actor Actor, in UpdateUserInput) (*model.User, error)
	Delete(ctx context.Context, actor Actor, id string) error
	Get(ctx context.Context, id string) (*model.User, error)
	List(ctx context.Context, n *pagination.Normalized) ([]model.User, *pagination.Meta, map[string]any, error)
}

// userService implements UserService.
type userService struct {
	db    *gorm.DB
	repo  *repository.Repo[model.User]
	rbac  RBACService
	audit *AuditService
}

// NewUserService wires the user domain.
func NewUserService(db *gorm.DB, repo *repository.Repo[model.User],
	rbac RBACService, audit *AuditService) UserService {
	return &userService{db: db, repo: repo, rbac: rbac, audit: audit}
}

func (s *userService) Create(ctx context.Context, actor Actor, in CreateUserInput) (*model.User, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if _, err := s.repo.FirstWhere(ctx, "email", email); err == nil {
		return nil, apperror.Conflict("email already registered")
	}
	hash, err := crypto.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	encPhone, _ := crypto.Encrypt(in.Phone)
	user := &model.User{
		Email: email, PasswordHash: hash, Name: in.Name, Phone: encPhone,
		Status: model.UserActive,
	}
	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, err
	}
	if len(in.RoleIDs) > 0 {
		if err := s.rbac.AssignRoles(ctx, user.ID, in.RoleIDs); err != nil {
			return nil, err
		}
	}
	_ = s.audit.Record(ctx, actor, constant.ActionCreate, "user", user.ID, nil, user)
	return s.repo.GetByID(ctx, user.ID)
}

func (s *userService) Update(ctx context.Context, actor Actor, in UpdateUserInput) (*model.User, error) {
	before, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	beforeSnapshot := *before // pre-update state for the audit trail
	if in.Name != "" {
		before.Name = in.Name
	}
	if in.Phone != "" {
		enc, _ := crypto.Encrypt(in.Phone)
		before.Phone = enc
	}
	if in.Status != "" {
		if in.Status != model.UserActive && in.Status != model.UserLocked {
			return nil, apperror.BadRequest("invalid status")
		}
		before.Status = in.Status
	}
	if err := s.repo.Save(ctx, before); err != nil {
		return nil, err
	}
	if in.RoleIDs != nil {
		if err := s.rbac.AssignRoles(ctx, in.ID, in.RoleIDs); err != nil {
			return nil, err
		}
	}
	_ = s.audit.Change(ctx, actor, "user", in.ID, &beforeSnapshot, before)
	return s.repo.GetByID(ctx, in.ID)
}

func (s *userService) Delete(ctx context.Context, actor Actor, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.rbac.InvalidateUser(ctx, id)
	_ = s.audit.Record(ctx, actor, constant.ActionDelete, "user", id, nil, nil)
	return nil
}

func (s *userService) Get(ctx context.Context, id string) (*model.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *userService) List(ctx context.Context, n *pagination.Normalized) ([]model.User, *pagination.Meta, map[string]any, error) {
	return s.repo.List(ctx, repository.ListQuery{P: n})
}

var _ UserService = (*userService)(nil)
