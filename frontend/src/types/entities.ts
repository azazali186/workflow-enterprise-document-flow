/** Entity shapes — mirrors backend/internal/model. */

export interface BaseEntity {
  id: string;
  created_at: string;
  updated_at: string;
}

export interface Role {
  id: string;
  code: string;
  name: string;
  description?: string;
  is_system: boolean;
  permissions?: Permission[];
  created_at: string;
  updated_at: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  phone?: string;
  status: 'active' | 'locked' | 'pending';
  avatar_url?: string;
  last_login_at?: string;
  roles?: Role[];
  created_at: string;
  updated_at: string;
}

export interface Permission {
  id: string;
  name: string;
  route: string; // "POST /api/v1/users/list"
  path: string;
  method: string;
  service: string;
  created_at: string;
  updated_at: string;
}

export interface Document {
  id: string;
  document_number: string;
  title: string;
  description?: string;
  category_id?: string;
  owner_id: string;
  status: DocumentStatus;
  file_name?: string;
  mime_type?: string;
  size_bytes?: number;
  content_hash?: string;
  meta?: Record<string, unknown>;
  tags?: string[];
  verified_at?: string;
  approved_at?: string;
  archived_at?: string;
  created_at: string;
  updated_at: string;
}

export type DocumentStatus =
  | 'draft'
  | 'pending_verification'
  | 'verified'
  | 'rejected'
  | 'approved'
  | 'archived';

export interface Category {
  id: string;
  name: string;
  slug: string;
  description?: string;
  parent_id?: string;
  sort_order: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface AuditLog {
  id: string;
  actor_id?: string;
  actor_email?: string;
  action: string;
  entity: string;
  entity_id?: string;
  before_data?: string;
  after_data?: string;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
  updated_at: string;
}

export interface LoginLog {
  id: string;
  user_id?: string;
  email: string;
  status: 'success' | 'failure';
  failure_reason?: string;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
  updated_at: string;
}

/** Auth result returned by login/register/refresh. */
export interface AuthResult {
  token: string;
  expires_at: string;
  /** Double-submit CSRF value bound to the token; echoed in X-CSRF-Token. */
  csrf?: string;
  user: User;
}

/** Approval chain step (backend model.Approval). */
export interface Approval {
  id: string;
  document_id: string;
  level: number;
  approver_id: string;
  status: 'pending' | 'approved' | 'rejected';
  comment?: string;
  requested_by?: string;
  decided_at?: string;
  created_at: string;
  updated_at: string;
}

/** Document verification request (backend model.Verification). */
export interface Verification {
  id: string;
  document_id: string;
  requested_by?: string;
  verified_by?: string;
  status: 'pending' | 'verified' | 'rejected';
  method?: string;
  notes?: string;
  result?: Record<string, unknown>;
  verified_at?: string;
  created_at: string;
  updated_at: string;
}

/** Document template (backend model.Template). */
export interface Template {
  id: string;
  name: string;
  slug: string;
  description?: string;
  category_id?: string;
  content?: string;
  version: number;
  is_active: boolean;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

/** Stored file record (backend model.Storage). object_key is encrypted on the server. */
export interface StorageRecord {
  id: string;
  document_id: string;
  provider: string;
  bucket?: string;
  file_name?: string;
  mime_type?: string;
  size_bytes: number;
  checksum?: string;
  status: string;
  stored_at?: string;
  created_at: string;
  updated_at: string;
}

/** Row-level document access grant (backend model.Access). */
export interface Access {
  id: string;
  document_id: string;
  user_id?: string;
  role_id?: string;
  permission: 'read' | 'write' | 'approve';
  granted_by?: string;
  revoked_at?: string;
  created_at: string;
  updated_at: string;
}

/** Document version snapshot (backend model.Version). */
export interface Version {
  id: string;
  document_id: string;
  version_number: number;
  change_summary?: string;
  snapshot?: Record<string, unknown>;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

/** Dashboard aggregate payload (backend reports/dashboard). */
export interface DashboardData {
  documents: Record<string, number>;
  verifications: Record<string, number>;
  approvals: Record<string, number>;
  storages: Record<string, number>;
  users: Record<string, number>;
  total_storage_bytes: number;
  documents_per_day: Record<string, number>;
  recent_activity: AuditLog[];
  pending_approvals: number;
}

/** Workflow analytics payload. */
export interface WorkflowData {
  funnel: Record<string, number>;
  pending_verifications: number;
  pending_approvals: number;
}
