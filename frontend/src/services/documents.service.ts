import { del, patch, post } from './http';
import type { ListRequest, PageResult } from '@/types/api';
import type { Document } from '@/types/entities';

export interface CreateDocumentInput {
  title: string;
  description?: string;
  category_id?: string;
  tags?: string[];
  meta?: Record<string, unknown>;
}

export interface UpdateDocumentInput {
  id: string;
  title?: string;
  description?: string;
  category_id?: string;
  tags?: string[];
  meta?: Record<string, unknown>;
  status?: string;
}

export const documentsService = {
  list(req: ListRequest, signal?: AbortSignal): Promise<PageResult<Document>> {
    return post<PageResult<Document>>('/api/v1/documents/list', req, { signal });
  },

  get(id: string): Promise<Document> {
    return post<Document>('/api/v1/documents/get', { id });
  },

  create(input: CreateDocumentInput): Promise<Document> {
    return post<Document>('/api/v1/documents/create', input);
  },

  update(input: UpdateDocumentInput): Promise<Document> {
    return patch<Document>('/api/v1/documents/update', input);
  },

  remove(id: string): Promise<void> {
    return del<void>('/api/v1/documents/delete', { id });
  },
};
