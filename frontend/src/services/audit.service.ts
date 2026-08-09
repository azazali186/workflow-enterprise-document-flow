import { post } from './http';
import type { ListRequest, PageResult } from '@/types/api';
import type { AuditLog, LoginLog } from '@/types/entities';

export const auditService = {
  list(req: ListRequest, signal?: AbortSignal): Promise<PageResult<AuditLog>> {
    return post<PageResult<AuditLog>>('/api/v1/audit-logs/list', req, { signal });
  },
};

export const loginLogService = {
  list(req: ListRequest, signal?: AbortSignal): Promise<PageResult<LoginLog>> {
    return post<PageResult<LoginLog>>('/api/v1/login-logs/list', req, { signal });
  },
};
