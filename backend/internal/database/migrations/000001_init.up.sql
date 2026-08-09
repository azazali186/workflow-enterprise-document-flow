-- 000001_init.up.sql
-- Initial DocuFlow schema. Mirrors the GORM models (snake_case, UUID v7 PKs,
-- soft delete via deleted_at). Versioned so production schema changes are
-- reviewable and reversible instead of AutoMigrate's implicit diffing.

CREATE TABLE IF NOT EXISTS users (
    id            uuid PRIMARY KEY,
    email         varchar(255) NOT NULL,
    password_hash varchar(255) NOT NULL,
    name          varchar(120),
    phone         varchar(255),
    status        varchar(20)  NOT NULL DEFAULT 'active',
    avatar_url    varchar(500),
    last_login_at timestamptz,
    created_at    timestamptz  NOT NULL,
    updated_at    timestamptz  NOT NULL,
    deleted_at    timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users (status);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at);

CREATE TABLE IF NOT EXISTS roles (
    id          uuid PRIMARY KEY,
    code        varchar(64)  NOT NULL,
    name        varchar(120) NOT NULL,
    description varchar(500),
    is_system   boolean      NOT NULL DEFAULT false,
    created_at  timestamptz  NOT NULL,
    updated_at  timestamptz  NOT NULL,
    deleted_at  timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_code ON roles (code);
CREATE INDEX IF NOT EXISTS idx_roles_deleted_at ON roles (deleted_at);
CREATE INDEX IF NOT EXISTS idx_roles_created_at ON roles (created_at);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id uuid NOT NULL,
    role_id uuid NOT NULL,
    PRIMARY KEY (user_id, role_id),
    CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS permissions (
    id         uuid PRIMARY KEY,
    name       varchar(120) NOT NULL,
    route      varchar(255) NOT NULL,
    path       varchar(255) NOT NULL,
    method     varchar(10)  NOT NULL,
    service    varchar(64)  NOT NULL DEFAULT 'api-gateway',
    created_at timestamptz  NOT NULL,
    updated_at timestamptz  NOT NULL,
    deleted_at timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_permissions_route ON permissions (route);
CREATE INDEX IF NOT EXISTS idx_permissions_path ON permissions (path);
CREATE INDEX IF NOT EXISTS idx_permissions_deleted_at ON permissions (deleted_at);
CREATE INDEX IF NOT EXISTS idx_permissions_created_at ON permissions (created_at);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       uuid NOT NULL,
    permission_id uuid NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE,
    CONSTRAINT fk_role_permissions_perm FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS categories (
    id          uuid PRIMARY KEY,
    name        varchar(120) NOT NULL,
    slug        varchar(120) NOT NULL,
    description varchar(500),
    parent_id   uuid,
    sort_order  integer      NOT NULL DEFAULT 0,
    is_active   boolean      NOT NULL DEFAULT false,
    created_at  timestamptz  NOT NULL,
    updated_at  timestamptz  NOT NULL,
    deleted_at  timestamptz,
    CONSTRAINT fk_categories_parent FOREIGN KEY (parent_id) REFERENCES categories (id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_slug ON categories (slug);
CREATE INDEX IF NOT EXISTS idx_categories_name ON categories (name);
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories (parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_is_active ON categories (is_active);
CREATE INDEX IF NOT EXISTS idx_categories_deleted_at ON categories (deleted_at);
CREATE INDEX IF NOT EXISTS idx_categories_created_at ON categories (created_at);

CREATE TABLE IF NOT EXISTS documents (
    id              uuid PRIMARY KEY,
    document_number varchar(64)  NOT NULL,
    title           varchar(255) NOT NULL,
    description     text,
    category_id     uuid,
    owner_id        uuid         NOT NULL,
    status          varchar(32)  NOT NULL DEFAULT 'draft',
    file_id         uuid,
    file_name       varchar(255),
    mime_type       varchar(120),
    size_bytes      bigint       NOT NULL DEFAULT 0,
    content_hash    varchar(64),
    meta            jsonb,
    tags            jsonb,
    verified_at     timestamptz,
    approved_at     timestamptz,
    archived_at     timestamptz,
    created_at      timestamptz  NOT NULL,
    updated_at      timestamptz  NOT NULL,
    deleted_at      timestamptz,
    CONSTRAINT fk_documents_owner FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT fk_documents_category FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_document_number ON documents (document_number);
CREATE INDEX IF NOT EXISTS idx_documents_category_id ON documents (category_id);
CREATE INDEX IF NOT EXISTS idx_documents_owner_id ON documents (owner_id);
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents (status);
CREATE INDEX IF NOT EXISTS idx_documents_deleted_at ON documents (deleted_at);
CREATE INDEX IF NOT EXISTS idx_documents_created_at ON documents (created_at);

CREATE TABLE IF NOT EXISTS versions (
    id             uuid PRIMARY KEY,
    document_id    uuid NOT NULL,
    version_number integer NOT NULL,
    change_summary text,
    snapshot       jsonb,
    created_by     uuid,
    created_at     timestamptz NOT NULL,
    updated_at     timestamptz NOT NULL,
    deleted_at     timestamptz,
    CONSTRAINT fk_versions_document FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT fk_versions_created_by FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_versions_document_id ON versions (document_id);
CREATE INDEX IF NOT EXISTS idx_versions_version_number ON versions (version_number);
CREATE INDEX IF NOT EXISTS idx_versions_created_by ON versions (created_by);
CREATE INDEX IF NOT EXISTS idx_versions_deleted_at ON versions (deleted_at);
CREATE INDEX IF NOT EXISTS idx_versions_created_at ON versions (created_at);

CREATE TABLE IF NOT EXISTS verifications (
    id           uuid PRIMARY KEY,
    document_id  uuid NOT NULL,
    requested_by uuid,
    verified_by  uuid,
    status       varchar(20) NOT NULL DEFAULT 'pending',
    method       varchar(32) NOT NULL DEFAULT 'manual',
    notes        text,
    result       jsonb,
    verified_at  timestamptz,
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL,
    deleted_at   timestamptz,
    CONSTRAINT fk_verifications_document FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT fk_verifications_requested_by FOREIGN KEY (requested_by) REFERENCES users (id) ON DELETE SET NULL,
    CONSTRAINT fk_verifications_verified_by FOREIGN KEY (verified_by) REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_verifications_document_id ON verifications (document_id);
CREATE INDEX IF NOT EXISTS idx_verifications_requested_by ON verifications (requested_by);
CREATE INDEX IF NOT EXISTS idx_verifications_verified_by ON verifications (verified_by);
CREATE INDEX IF NOT EXISTS idx_verifications_status ON verifications (status);
CREATE INDEX IF NOT EXISTS idx_verifications_deleted_at ON verifications (deleted_at);
CREATE INDEX IF NOT EXISTS idx_verifications_created_at ON verifications (created_at);

CREATE TABLE IF NOT EXISTS approvals (
    id           uuid PRIMARY KEY,
    document_id  uuid NOT NULL,
    level        integer NOT NULL,
    approver_id  uuid NOT NULL,
    status       varchar(20) NOT NULL DEFAULT 'pending',
    comment      text,
    requested_by uuid,
    decided_at   timestamptz,
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL,
    deleted_at   timestamptz,
    CONSTRAINT fk_approvals_document FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT fk_approvals_approver FOREIGN KEY (approver_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT fk_approvals_requested_by FOREIGN KEY (requested_by) REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_approvals_document_id ON approvals (document_id);
CREATE INDEX IF NOT EXISTS idx_approvals_approver_id ON approvals (approver_id);
CREATE INDEX IF NOT EXISTS idx_approvals_status ON approvals (status);
CREATE INDEX IF NOT EXISTS idx_approvals_requested_by ON approvals (requested_by);
CREATE INDEX IF NOT EXISTS idx_approvals_deleted_at ON approvals (deleted_at);
CREATE INDEX IF NOT EXISTS idx_approvals_created_at ON approvals (created_at);

CREATE TABLE IF NOT EXISTS storages (
    id          uuid PRIMARY KEY,
    document_id uuid NOT NULL,
    provider    varchar(32) NOT NULL DEFAULT 'local',
    bucket      varchar(120),
    object_key  varchar(500),
    file_name   varchar(255),
    mime_type   varchar(120),
    size_bytes  bigint NOT NULL DEFAULT 0,
    checksum    varchar(64),
    status      varchar(20) NOT NULL DEFAULT 'stored',
    stored_at   timestamptz,
    created_at  timestamptz NOT NULL,
    updated_at  timestamptz NOT NULL,
    deleted_at  timestamptz,
    CONSTRAINT fk_storages_document FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_storages_document_id ON storages (document_id);
CREATE INDEX IF NOT EXISTS idx_storages_status ON storages (status);
CREATE INDEX IF NOT EXISTS idx_storages_deleted_at ON storages (deleted_at);
CREATE INDEX IF NOT EXISTS idx_storages_created_at ON storages (created_at);

CREATE TABLE IF NOT EXISTS templates (
    id          uuid PRIMARY KEY,
    name        varchar(120) NOT NULL,
    slug        varchar(120) NOT NULL,
    description varchar(500),
    category_id uuid,
    content     text,
    version     integer NOT NULL DEFAULT 1,
    is_active   boolean NOT NULL DEFAULT false,
    created_by  uuid,
    created_at  timestamptz NOT NULL,
    updated_at  timestamptz NOT NULL,
    deleted_at  timestamptz,
    CONSTRAINT fk_templates_category FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE SET NULL,
    CONSTRAINT fk_templates_created_by FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_templates_slug ON templates (slug);
CREATE INDEX IF NOT EXISTS idx_templates_name ON templates (name);
CREATE INDEX IF NOT EXISTS idx_templates_category_id ON templates (category_id);
CREATE INDEX IF NOT EXISTS idx_templates_is_active ON templates (is_active);
CREATE INDEX IF NOT EXISTS idx_templates_created_by ON templates (created_by);
CREATE INDEX IF NOT EXISTS idx_templates_deleted_at ON templates (deleted_at);
CREATE INDEX IF NOT EXISTS idx_templates_created_at ON templates (created_at);

CREATE TABLE IF NOT EXISTS accesses (
    id          uuid PRIMARY KEY,
    document_id uuid NOT NULL,
    user_id     uuid,
    role_id     uuid,
    permission  varchar(20) NOT NULL DEFAULT 'read',
    granted_by  uuid,
    revoked_at  timestamptz,
    created_at  timestamptz NOT NULL,
    updated_at  timestamptz NOT NULL,
    deleted_at  timestamptz,
    CONSTRAINT fk_accesses_document FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT fk_accesses_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT fk_accesses_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE RESTRICT,
    CONSTRAINT fk_accesses_granted_by FOREIGN KEY (granted_by) REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_accesses_document_id ON accesses (document_id);
CREATE INDEX IF NOT EXISTS idx_accesses_user_id ON accesses (user_id);
CREATE INDEX IF NOT EXISTS idx_accesses_role_id ON accesses (role_id);
CREATE INDEX IF NOT EXISTS idx_accesses_granted_by ON accesses (granted_by);
CREATE INDEX IF NOT EXISTS idx_accesses_revoked_at ON accesses (revoked_at);
CREATE INDEX IF NOT EXISTS idx_accesses_deleted_at ON accesses (deleted_at);
CREATE INDEX IF NOT EXISTS idx_accesses_created_at ON accesses (created_at);

CREATE TABLE IF NOT EXISTS login_logs (
    id             uuid PRIMARY KEY,
    user_id        uuid,
    email          varchar(255),
    status         varchar(20) NOT NULL,
    failure_reason varchar(255),
    ip_address     varchar(64),
    user_agent     varchar(500),
    created_at     timestamptz NOT NULL,
    updated_at     timestamptz NOT NULL,
    deleted_at     timestamptz,
    CONSTRAINT fk_login_logs_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_login_logs_user_id ON login_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_login_logs_email ON login_logs (email);
CREATE INDEX IF NOT EXISTS idx_login_logs_status ON login_logs (status);
CREATE INDEX IF NOT EXISTS idx_login_logs_deleted_at ON login_logs (deleted_at);
CREATE INDEX IF NOT EXISTS idx_login_logs_created_at ON login_logs (created_at);

CREATE TABLE IF NOT EXISTS audit_logs (
    id           uuid PRIMARY KEY,
    actor_id     uuid,
    actor_email  varchar(255),
    action       varchar(32) NOT NULL,
    entity       varchar(64) NOT NULL,
    entity_id    varchar(64),
    before_data  text,
    after_data   text,
    ip_address   varchar(64),
    user_agent   varchar(500),
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL,
    deleted_at   timestamptz,
    CONSTRAINT fk_audit_logs_actor FOREIGN KEY (actor_id) REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_id ON audit_logs (actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_email ON audit_logs (actor_email);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs (action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs (entity);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_id ON audit_logs (entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_deleted_at ON audit_logs (deleted_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at);

CREATE TABLE IF NOT EXISTS outbox_messages (
    id             uuid PRIMARY KEY,
    aggregate_type varchar(64),
    aggregate_id   varchar(64),
    event_type     varchar(64),
    payload        text,
    status         varchar(16) NOT NULL DEFAULT 'pending',
    attempts       integer     NOT NULL DEFAULT 0,
    last_error     text,
    published_at   timestamptz,
    created_at     timestamptz NOT NULL,
    updated_at     timestamptz NOT NULL,
    deleted_at     timestamptz
);
CREATE INDEX IF NOT EXISTS idx_outbox_messages_aggregate_type ON outbox_messages (aggregate_type);
CREATE INDEX IF NOT EXISTS idx_outbox_messages_aggregate_id ON outbox_messages (aggregate_id);
CREATE INDEX IF NOT EXISTS idx_outbox_messages_event_type ON outbox_messages (event_type);
CREATE INDEX IF NOT EXISTS idx_outbox_messages_status ON outbox_messages (status);
CREATE INDEX IF NOT EXISTS idx_outbox_messages_deleted_at ON outbox_messages (deleted_at);
CREATE INDEX IF NOT EXISTS idx_outbox_messages_created_at ON outbox_messages (created_at);
