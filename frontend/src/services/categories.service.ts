import { del, patch, post } from './http';
import type { ListRequest, PageResult } from '@/types/api';
import type { Category } from '@/types/entities';

export interface CreateCategoryInput {
  name: string;
  slug: string;
  description?: string;
  parent_id?: string;
  sort_order?: number;
}

export interface UpdateCategoryInput {
  id: string;
  name?: string;
  description?: string;
  parent_id?: string;
  sort_order?: number;
  is_active?: boolean;
}

export const categoriesService = {
  list(req: ListRequest, signal?: AbortSignal): Promise<PageResult<Category>> {
    return post<PageResult<Category>>('/api/v1/categories/list', req, { signal });
  },

  create(input: CreateCategoryInput): Promise<Category> {
    return post<Category>('/api/v1/categories/create', input);
  },

  update(input: UpdateCategoryInput): Promise<Category> {
    return patch<Category>('/api/v1/categories/update', input);
  },

  remove(id: string): Promise<void> {
    return del<void>('/api/v1/categories/delete', { id });
  },
};
