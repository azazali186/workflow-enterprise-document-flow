import { describe, expect, it } from 'vitest';
import { groupByEntity } from './permissionGroups';
import type { Permission } from '@/types/entities';

function perm(id: string, method: string, path: string): Permission {
  return {
    id,
    method,
    path,
    name: `${method} ${path}`,
    route: `${method} ${path}`,
    service: 'x',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

describe('groupByEntity', () => {
  it('groups by the first path segment', () => {
    const groups = groupByEntity([
      perm('1', 'POST', '/api/v1/users/create'),
      perm('2', 'POST', '/api/v1/roles/create'),
      perm('3', 'PATCH', '/api/v1/users/update'),
    ]);
    expect(groups.map((g) => g.entity)).toEqual(['roles', 'users']);
    expect(groups[1].perms.map((p) => p.id)).toEqual(['1', '3']);
  });

  it('sorts paths within a group and groups alphabetically', () => {
    const groups = groupByEntity([
      perm('1', 'POST', '/api/v1/documents/delete'),
      perm('2', 'POST', '/api/v1/documents/create'),
      perm('3', 'POST', '/api/v1/audit-logs/list'),
    ]);
    expect(groups[0].entity).toBe('audit-logs');
    expect(groups[1].perms.map((p) => p.path)).toEqual([
      '/api/v1/documents/create',
      '/api/v1/documents/delete',
    ]);
  });

  it('falls back to "other" for paths without an entity segment', () => {
    const groups = groupByEntity([perm('1', 'GET', '/api/v1/')]);
    expect(groups[0].entity).toBe('other');
  });

  it('handles an empty list', () => {
    expect(groupByEntity([])).toEqual([]);
  });
});
