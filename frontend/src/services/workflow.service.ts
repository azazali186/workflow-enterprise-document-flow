import { del, patch, post } from './http';
import type { ListRequest, PageResult } from '@/types/api';
import type { Access, Approval, StorageRecord, Template, Verification, Version } from '@/types/entities';

export const approvalsService = {
  list(req: ListRequest, signal?: AbortSignal): Promise<PageResult<Approval>> {
    return post<PageResult<Approval>>('/api/v1/approvals/list', req, { signal });
  },
  decide(input: { approval_id: string; decision: 'approved' | 'rejected'; comment?: string }): Promise<Approval> {
    return post<Approval>('/api/v1/approvals/decide', input);
  },
  create(input: { document_id: string; approver_ids: string[] }): Promise<unknown> {
    return post('/api/v1/approvals/create', input);
  },
};

export const verificationsService = {
  list(req: ListRequest, signal?: AbortSignal): Promise<PageResult<Verification>> {
    return post<PageResult<Verification>>('/api/v1/verifications/list', req, { signal });
  },
  decide(input: { verification_id: string; decision: 'verified' | 'rejected'; notes?: string }): Promise<Verification> {
    return post<Verification>('/api/v1/verifications/decide', input);
  },
  create(input: { document_id: string; method?: string; notes?: string }): Promise<unknown> {
    return post('/api/v1/verifications/create', input);
  },
};

export const templatesService = {
  list(req: ListRequest, signal?: AbortSignal): Promise<PageResult<Template>> {
    return post<PageResult<Template>>('/api/v1/templates/list', req, { signal });
  },
  create(input: { name: string; slug: string; description?: string; content?: string }): Promise<Template> {
    return post<Template>('/api/v1/templates/create', input);
  },
  update(input: { id: string; name?: string; description?: string; content?: string; is_active?: boolean }): Promise<Template> {
    return patch<Template>('/api/v1/templates/update', input);
  },
  remove(id: string): Promise<void> {
    return del<void>('/api/v1/templates/delete', { id });
  },
};

export const storagesService = {
  list(req: ListRequest, signal?: AbortSignal): Promise<PageResult<StorageRecord>> {
    return post<PageResult<StorageRecord>>('/api/v1/storages/list', req, { signal });
  },
};

export const accessesService = {
  list(req: ListRequest, signal?: AbortSignal): Promise<PageResult<Access>> {
    return post<PageResult<Access>>('/api/v1/accesses/list', req, { signal });
  },
  grant(input: { document_id: string; user_id?: string; role_id?: string; permission: 'read' | 'write' | 'approve' }): Promise<unknown> {
    return post('/api/v1/accesses/grant', input);
  },
  revoke(accessId: string): Promise<unknown> {
    return post('/api/v1/accesses/revoke', { access_id: accessId });
  },
};

export const versionsService = {
  list(documentId: string, req: ListRequest, signal?: AbortSignal): Promise<PageResult<Version>> {
    return post<PageResult<Version>>('/api/v1/versions/list', { ...req, document_id: documentId }, { signal });
  },
};
