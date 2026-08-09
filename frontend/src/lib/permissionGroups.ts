import type { Permission } from '@/types/entities';

export interface PermissionGroup {
  entity: string;
  perms: Permission[];
}

/** Groups permissions by their path prefix for a scannable list. */
export function groupByEntity(perms: Permission[]): PermissionGroup[] {
  const map = new Map<string, Permission[]>();
  for (const p of perms) {
    const parts = p.path.replace('/api/v1/', '').split('/');
    const entity = parts[0] || 'other';
    map.set(entity, [...(map.get(entity) ?? []), p]);
  }
  return [...map.entries()]
    .map(([entity, perms]) => ({ entity, perms: [...perms].sort((a, b) => a.path.localeCompare(b.path)) }))
    .sort((a, b) => a.entity.localeCompare(b.entity));
}
