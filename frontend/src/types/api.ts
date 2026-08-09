/** Shared API contract types — mirrors backend/internal/pkg/response and pagination. */

/** Unified envelope: HTTP is always 200; the business code lives in the body. */
export interface ApiEnvelope<T = unknown> {
  code: number;
  message: string;
  data?: T;
}

/** Cursor pagination metadata (backend pkg/pagination.Meta). */
export interface PaginationMeta {
  next_cursor: string;
  has_more: boolean;
  limit: number;
  returned_count: number;
  total_count: number;
}

/** List request body accepted by every list endpoint. */
export interface ListRequest {
  cursor?: string;
  limit?: number;
  filters?: Record<string, unknown>;
  sort_by?: string;
  sort_dir?: 'asc' | 'desc';
  date_from?: string;
  date_to?: string;
  search?: string;
}

/** List response body (backend response.Page). */
export interface PageResult<T> {
  items: T[];
  pagination: PaginationMeta;
  summary?: Record<string, unknown>;
}

/** Normalized error thrown by the API client. */
export class ApiError extends Error {
  readonly code: number;
  readonly status: number;

  constructor(message: string, code: number, status = 200) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
  }
}
