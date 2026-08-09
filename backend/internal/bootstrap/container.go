// Package bootstrap wires the application: infrastructure, repositories,
// services and handlers (manual dependency injection, no framework magic).
package bootstrap

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/config"
	"github.com/aeroxe/docu-flow/backend/internal/database"
	"github.com/aeroxe/docu-flow/backend/internal/handler"
	"github.com/aeroxe/docu-flow/backend/internal/middleware"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/objectstore"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/crypto"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/jwt"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/lock"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/outbox"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"github.com/aeroxe/docu-flow/backend/internal/router"
	"github.com/aeroxe/docu-flow/backend/internal/saga"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/aeroxe/docu-flow/backend/internal/worker"
	"github.com/aeroxe/docu-flow/backend/internal/ws"
	"gorm.io/gorm"
)

// Container holds every wired dependency.
type Container struct {
	Cfg        *config.Config
	DB         *gorm.DB
	Cache      *cache.Client
	Lock       *lock.Lock
	Dispatcher *outbox.Dispatcher
	Worker     *worker.Worker
	Hub        *ws.Hub
	RBAC       service.RBACService
	Handlers   *router.Handlers
}

// New boots the whole dependency graph.
func New(ctx context.Context, cfg *config.Config) (*Container, error) {
	if err := crypto.Init(cfg.EncryptionKey); err != nil {
		return nil, err
	}
	jwt.Init(cfg.JWTSecret, cfg.JWTExpiry)

	db, err := database.Init(cfg)
	if err != nil {
		return nil, err
	}
	if err := database.MigrateVersioned(db, cfg); err != nil {
		return nil, err
	}
	if err := database.SeedRBAC(db, cfg); err != nil {
		return nil, err
	}
	if err := database.InitRedis(ctx, cfg); err != nil {
		return nil, err
	}
	if err := database.InitNATS(cfg); err != nil {
		return nil, err
	}

	store, err := objectstore.New(cfg)
	if err != nil {
		return nil, err
	}

	lk := lock.New(database.RDB)
	audit := service.NewAuditService(db)

	// Repositories (one per entity).
	userRepo := repository.NewUserRepo(db)
	roleRepo := repository.NewRoleRepo(db)
	permRepo := repository.NewPermissionRepo(db)
	docRepo := repository.NewDocumentRepo(db)
	versionRepo := repository.NewVersionRepo(db)
	categoryRepo := repository.NewCategoryRepo(db)
	templateRepo := repository.NewTemplateRepo(db)
	storageRepo := repository.NewStorageRepo(db)
	verifRepo := repository.NewVerificationRepo(db)
	approvalRepo := repository.NewApprovalRepo(db)
	accessRepo := repository.NewAccessRepo(db)
	loginLogRepo := repository.NewLoginLogRepo(db)
	auditLogRepo := repository.NewAuditLogRepo(db)

	// Pipeline integrations (virus scanning + search indexing). Defaults to
	// noops; enable via VIRUS_SCANNER=clamav and INDEXER=opensearch.
	var scanner worker.Scanner = worker.NoopScanner{}
	if cfg.VirusScanner == "clamav" {
		scanner = worker.NewClamAVScanner(cfg.ClamAVAddr, cfg.ClamAVTimeout)
	}
	var indexer service.Indexer = worker.NoopIndexer{}
	if cfg.Indexer == "opensearch" {
		indexer = worker.NewOpenSearchIndexer(cfg.OpenSearchURL, cfg.OpenSearchIndex,
			cfg.OpenSearchUser, cfg.OpenSearchPass)
	}

	// Services.
	rbacSvc := service.NewRBACService(db, database.Cache, userRepo)
	authSvc := service.NewAuthService(db, userRepo, roleRepo, loginLogRepo, database.Cache, audit, cfg.JWTExpiry)
	userSvc := service.NewUserService(db, userRepo, rbacSvc, audit)
	sagas := saga.New(database.Cache, db)
	docSvc := service.NewDocumentService(db, docRepo, versionRepo, audit, database.Cache, sagas, indexer)
	verifSvc := service.NewVerificationService(db, verifRepo, docRepo, audit, lk, sagas)
	approvalSvc := service.NewApprovalService(db, approvalRepo, docRepo, audit, lk, sagas)
	storageSvc := service.NewStorageService(storageRepo, docRepo, audit)
	reportSvc := service.NewReportService(db)
	searchSvc := service.NewSearchService(db, docRepo)
	analyticsSvc := service.NewAnalyticsService(db)
	optionsSvc := service.NewOptionsService(db)

	// Generic CRUD services.
	roleCrud := &service.CrudService[model.Role]{Repo: roleRepo, Audit: audit, Entity: "role"}
	permCrud := &service.CrudService[model.Permission]{Repo: permRepo, Audit: audit, Entity: "permission"}
	versionCrud := &service.CrudService[model.Version]{Repo: versionRepo, Audit: audit, Entity: "version"}
	categoryCrud := &service.CrudService[model.Category]{Repo: categoryRepo, Audit: audit, Entity: "category"}
	templateCrud := &service.CrudService[model.Template]{Repo: templateRepo, Audit: audit, Entity: "template"}
	accessCrud := &service.CrudService[model.Access]{Repo: accessRepo, Audit: audit, Entity: "access"}
	auditLogCrud := &service.CrudService[model.AuditLog]{Repo: auditLogRepo, Audit: audit, Entity: "audit_log"}
	loginLogCrud := &service.CrudService[model.LoginLog]{Repo: loginLogRepo, Audit: audit, Entity: "login_log"}

	disp := outbox.NewDispatcher(db, &natsPublisher{}, outbox.Options{
		Interval: cfg.OutboxInterval, Batch: cfg.OutboxBatch, MaxAttempts: cfg.OutboxMaxAttempts,
	})
	hub := ws.NewHub()
	svcWorker := worker.New(db, database.JS, database.Cache, sagas, hub,
		docRepo, storageRepo, approvalRepo, audit,
		worker.Options{Scanner: scanner, Indexer: indexer, Store: store})

	// Handlers.
	hs := &router.Handlers{
		Auth:         handler.NewAuthHandler(authSvc, cfg.IsProduction()),
		User:         handler.NewUserHandler(userSvc),
		Role:         handler.NewRoleHandler(roleCrud, rbacSvc),
		Permission:   handler.NewPermissionHandler(permCrud),
		Document:     handler.NewDocumentHandler(docSvc),
		Version:      handler.NewVersionHandler(versionCrud),
		Category:     handler.NewCategoryHandler(categoryCrud),
		Template:     handler.NewTemplateHandler(templateCrud),
		Storage:      handler.NewStorageHandler(storageSvc, store),
		Verification: handler.NewVerificationHandler(verifSvc),
		Approval:     handler.NewApprovalHandler(approvalSvc),
		Access:       handler.NewAccessHandler(accessCrud),
		AuditLog:     handler.NewAuditLogHandler(auditLogCrud),
		LoginLog:     handler.NewLoginLogHandler(loginLogCrud),
		Report:       handler.NewReportHandler(reportSvc),
		Search:       handler.NewSearchHandler(searchSvc),
		Analytics:    handler.NewAnalyticsHandler(analyticsSvc),
		Options:      handler.NewOptionsHandler(optionsSvc),
		Health:       handler.NewHealthHandler(),
		WS: ws.NewEndpoint(hub, database.Cache, cfg.JWTExpiry, cfg.CORSOrigins,
			ws.ActiveCheck(ActiveUserStatus(db))),
	}

	return &Container{
		Cfg: cfg, DB: db, Cache: database.Cache, Lock: lk, Dispatcher: disp,
		Worker: svcWorker, Hub: hub, RBAC: rbacSvc, Handlers: hs,
	}, nil
}

// ActiveUserStatus builds the account-status checker the HTTP auth middleware
// and the WebSocket endpoint share: active accounts only; any transient DB
// error fails closed.
func ActiveUserStatus(db *gorm.DB) middleware.UserStatusCheck {
	return func(ctx context.Context, userID string) (bool, error) {
		var u model.User
		err := db.WithContext(ctx).Select("status").First(&u, "id = ?", userID).Error
		if err != nil {
			return false, err
		}
		return u.Status == model.UserActive, nil
	}
}

// natsPublisher adapts JetStream to the outbox Publisher interface.
type natsPublisher struct{}

// Publish implements outbox.Publisher.
func (n *natsPublisher) Publish(subject string, data []byte) error {
	_, err := database.JS.Publish(subject, data)
	return err
}
