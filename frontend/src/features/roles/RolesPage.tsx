import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { KeyRound, Pencil, Plus, Trash2 } from 'lucide-react';
import { PageHeader } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { DataTable, PaginationBar, type Column } from '@/components/ui/DataTable';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { RoleFormModal } from './RoleFormModal';
import { PermissionsModal } from './PermissionsModal';
import { usePaginatedQuery } from '@/hooks/usePaginatedQuery';
import { useToast, errorMessage } from '@/hooks/useToast';
import { rolesService } from '@/services/roles.service';
import type { Role } from '@/types/entities';

export function RolesPage() {
  const qc = useQueryClient();
  const toast = useToast();
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Role | null>(null);
  const [permsRole, setPermsRole] = useState<Role | null>(null);
  const [deleting, setDeleting] = useState<Role | null>(null);

  const table = usePaginatedQuery<Role>(['roles'], (req, signal) => rolesService.list(req, signal), {
    defaultSortBy: 'name',
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => rolesService.remove(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['roles'] });
      toast.success('Role deleted');
      setDeleting(null);
    },
    onError: (err) => toast.error('Could not delete role', errorMessage(err)),
  });

  const columns: Column<Role>[] = [
    {
      key: 'name',
      header: 'Role',
      sortKey: 'name',
      cell: (r) => (
        <div>
          <p className="font-medium text-ink-900">{r.name}</p>
          <p className="text-xs text-ink-400">{r.code}</p>
        </div>
      ),
    },
    {
      key: 'description',
      header: 'Description',
      cell: (r) => <span className="text-[13px] text-ink-500">{r.description || '—'}</span>,
    },
    {
      key: 'permissions',
      header: 'Permissions',
      cell: (r) => <Badge tone="primary">{r.permissions?.length ?? 0} routes</Badge>,
    },
    {
      key: 'system',
      header: 'Type',
      cell: (r) => (r.is_system ? <Badge tone="neutral">System</Badge> : <Badge tone="success">Custom</Badge>),
    },
    {
      key: 'actions',
      header: '',
      className: 'w-32 text-right',
      cell: (r) => (
        <div className="flex justify-end gap-1">
          <button
            onClick={(e) => { e.stopPropagation(); setPermsRole(r); }}
            title="Manage permissions"
            aria-label={`Manage permissions for ${r.name}`}
            className="rounded-lg p-2 text-ink-400 transition-colors hover:bg-primary-50 hover:text-primary-600 cursor-pointer"
          >
            <KeyRound className="size-4" />
          </button>
          <button
            onClick={(e) => { e.stopPropagation(); setEditing(r); setFormOpen(true); }}
            aria-label={`Edit ${r.name}`}
            className="rounded-lg p-2 text-ink-400 transition-colors hover:bg-primary-50 hover:text-primary-600 cursor-pointer"
          >
            <Pencil className="size-4" />
          </button>
          {!r.is_system && (
            <button
              onClick={(e) => { e.stopPropagation(); setDeleting(r); }}
              aria-label={`Delete ${r.name}`}
              className="rounded-lg p-2 text-ink-400 transition-colors hover:bg-danger-50 hover:text-danger-600 cursor-pointer"
            >
              <Trash2 className="size-4" />
            </button>
          )}
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-5 animate-fade-up">
      <PageHeader
        title="Roles & permissions"
        description="Control what each role can do across the API."
        actions={
          <Button icon={<Plus className="size-4" />} onClick={() => { setEditing(null); setFormOpen(true); }}>
            New role
          </Button>
        }
      />

      <DataTable<Role>
        columns={columns}
        rows={table.rows}
        meta={table.meta}
        loading={table.isLoading}
        error={table.error}
        onRetry={() => void table.refetch()}
        sortBy={table.sortBy}
        sortDir={table.sortDir}
        onSort={table.toggleSort}
        rowKey={(r) => r.id}
        emptyTitle="No roles yet"
        emptyDescription="Roles group permissions and are assigned to users."
        summary={table.summary}
        footer={
          <PaginationBar meta={table.meta} onNext={table.next} onPrev={table.prev} canPrev={table.canPrev} />
        }
      />

      <RoleFormModal open={formOpen} onClose={() => setFormOpen(false)} role={editing} />
      <PermissionsModal open={Boolean(permsRole)} onClose={() => setPermsRole(null)} role={permsRole} />

      <ConfirmDialog
        open={Boolean(deleting)}
        onCancel={() => setDeleting(null)}
        onConfirm={() => deleting && deleteMutation.mutate(deleting.id)}
        loading={deleteMutation.isPending}
        title="Delete role"
        message={`Delete "${deleting?.name}"? Users holding this role will lose its permissions.`}
        confirmLabel="Delete role"
      />
    </div>
  );
}
