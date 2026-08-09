import { post } from './http';

/** Entities served by the shared options endpoint. */
export type OptionKind = 'users' | 'roles' | 'categories' | 'templates' | 'documents';

/** The universal {id, name} shape every dropdown consumes. */
export interface OptionItem {
  id: string;
  name: string;
}

export const optionsService = {
  /**
   * Fetches up to `limit` {id, name} options for a dropdown, filtered by an
   * optional case-insensitive search. Users are limited to active accounts.
   */
  list(kind: OptionKind, search?: string, limit = 20, signal?: AbortSignal): Promise<OptionItem[]> {
    return post<OptionItem[]>('/api/v1/options/list', {
      type: kind,
      search: search?.trim() || undefined,
      limit,
    }, { signal });
  },
};
