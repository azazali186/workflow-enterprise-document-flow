package database

import (
	"errors"
	"fmt"

	"github.com/aeroxe/docu-flow/backend/internal/config"
	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/crypto"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// builtinRoles defines roles created on first boot.
var builtinRoles = []model.Role{
	{Code: constant.RoleSuperAdmin, Name: "Super Admin", Description: "Full access to every route", IsSystem: true},
	{Code: constant.RoleAdmin, Name: "Administrator", Description: "Manages documents, users and workflows", IsSystem: true},
	{Code: constant.RoleUser, Name: "User", Description: "Standard document user", IsSystem: true},
}

// SeedRBAC creates built-in roles and the bootstrap admin account. It is
// idempotent and safe to run on every startup.
func SeedRBAC(db *gorm.DB, cfg *config.Config) error {
	if err := seedRoles(db); err != nil {
		return err
	}
	return seedAdmin(db, cfg)
}

func seedRoles(db *gorm.DB) error {
	for _, r := range builtinRoles {
		var existing model.Role
		err := db.Where("code = ?", r.Code).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&r).Error; err != nil {
				return err
			}
			logger.Info("seeded role", zap.String("code", r.Code))
			continue
		}
		if existing.Name != r.Name || existing.IsSystem != r.IsSystem {
			existing.Name = r.Name
			existing.IsSystem = r.IsSystem
			if err := db.Save(&existing).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// seedAdmin ensures the bootstrap admin account exists. It is keyed by the
// admin email rather than "database is empty", so a restored or migrated DB
// that already has users still gets its super admin (previously a DB with any
// user silently skipped the admin, leaving the system with no one who could
// manage it). ON CONFLICT DO NOTHING makes concurrent replica startups safe:
// the loser's insert is a no-op and it simply skips the role link the winner
// already created.
func seedAdmin(db *gorm.DB, cfg *config.Config) error {
	hash, err := crypto.HashPassword(cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	admin := model.User{
		Email:        cfg.AdminEmail,
		PasswordHash: hash,
		Name:         cfg.AdminName,
		Status:       model.UserActive,
	}
	return db.Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&admin)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // another replica seeded the admin concurrently
		}
		var superRole model.Role
		if err := tx.Where("code = ?", constant.RoleSuperAdmin).First(&superRole).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&model.UserRole{UserID: admin.ID, RoleID: superRole.ID}).Error; err != nil {
			return err
		}
		logger.Info("seeded bootstrap admin", zap.String("email", cfg.AdminEmail))
		return nil
	})
}
