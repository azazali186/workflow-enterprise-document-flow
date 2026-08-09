import { del, patch, post } from './http';
import type { ListRequest, PageResult } from '@/types/api';
import type { User } from '@/types/entities';

export interface CreateUserInput {
  email: string;
  password: string;
  name: string;
  phone?: string;
  role_ids?: string[];
}

export interface UpdateUserInput {
  id: string;
  name?: string;
  phone?: string;
  status?: string;
  role_ids?: string[];
}

export const usersService = {
  list(req: ListRequest, signal?: AbortSignal): Promise<PageResult<User>> {
    return post<PageResult<User>>('/api/v1/users/list', req, { signal });
  },

  get(id: string): Promise<User> {
    return post<User>('/api/v1/users/get', { id });
  },

  create(input: CreateUserInput): Promise<User> {
    return post<User>('/api/v1/users/create', input);
  },

  update(input: UpdateUserInput): Promise<User> {
    return patch<User>('/api/v1/users/update', input);
  },

  remove(id: string): Promise<void> {
    return del<void>('/api/v1/users/delete', { id });
  },
};
