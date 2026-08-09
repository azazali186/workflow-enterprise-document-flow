import { del, patch, post } from './http';
import type { ListRequest, PageResult } from '@/types/api';
import type { Permission, Role } from '@/types/entities';

export interface CreateRoleInput {
  code: string;
  name: string;
  description?: string;
}

export interface UpdateRoleInput {
  id: string;
  name?: string;
  description?: string;
}

export const rolesService = {
  list(req: ListRequest, signal?: AbortSignal): Promise<PageResult<Role>> {
    return post<PageResult<Role>>('/api/v1/roles/list', req, { signal });
  },

  get(id: string): Promise<Role> {
    return post<Role>('/api/v1/roles/get', { id });
  },

  create(input: CreateRoleInput): Promise<Role> {
    return post<Role>('/api/v1/roles/create', input);
  },

  update(input: UpdateRoleInput): Promise<Role> {
    return patch<Role>('/api/v1/roles/update', input);
  },

  remove(id: string): Promise<void> {
    return del<void>('/api/v1/roles/delete', { id });
  },

  assignPermissions(roleId: string, permissionIds: string[]): Promise<void> {
    return post<void>('/api/v1/roles/assign-permissions', {
      role_id: roleId,
      permission_ids: permissionIds,
    });
  },
};

export const permissionsService = {
  list(req: ListRequest, signal?: AbortSignal): Promise<PageResult<Permission>> {
    return post<PageResult<Permission>>('/api/v1/permissions/list', req, { signal });
  },
};
