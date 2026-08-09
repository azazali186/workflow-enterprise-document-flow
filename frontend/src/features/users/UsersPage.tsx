import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Pencil, Plus, Search, Trash2 } from 'lucide-react';
import { PageHeader } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Select } from '@/components/ui/Field';
import { Badge, statusTone } from '@/components/ui/Badge';
import { DataTable, PaginationBar, type Column } from '@/components/ui/DataTable';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { UserFormModal } from './UserFormModal';
import { Avatar } from '@/components/layout/Avatar';
import { usePaginatedQuery } from '@/hooks/usePaginatedQuery';
import { useToast, errorMessage } from '@/hooks/useToast';
import { usersService } from '@/services/users.service';
import { formatDateTime } from '@/lib/format';
import type { User } from '@/types/entities';

export function UsersPage() {
  const qc = useQueryClient();
  const toast = useToast();
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<User | null>(null);
  const [deleting, setDeleting] = useState<User | null>(null);

  const table = usePaginatedQuery<User>(['users'], (req, signal) => usersService.list(req, signal), {
    defaultSortBy: 'created_at',
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => usersService.remove(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['users'] });
      toast.success('User deleted');
      setDeleting(null);
    },
    onError: (err) => toast.error('Could not delete user', errorMessage(err)),
  });

  const columns: Column<User>[] = [
    {
      key: 'user',
      header: 'User',
      sortKey: 'name',
      cell: (u) => (
        <div className="flex items-center gap-3">
          <Avatar name={u.name || u.email} />
          <div className="min-w-0">
            <p className="truncate font-medium text-ink-900">{u.name || '—'}</p>
            <p className="truncate text-xs text-ink-400">{u.email}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'roles',
      header: 'Roles',
      cell: (u) => (
        <div className="flex flex-wrap gap-1.5">
          {u.roles?.length ? (
            u.roles.map((r) => <Badge key={r.id} tone={r.code === 'super_admin' ? 'danger' : 'primary'}>{r.name}</Badge>)
          ) : (
            <span className="text-xs text-ink-300">No roles</span>
          )}
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      sortKey: 'status',
      cell: (u) => <Badge tone={statusTone(u.status)} dot>{u.status}</Badge>,
    },
    {
      key: 'last_login',
      header: 'Last login',
      sortKey: 'last_login_at',
      cell: (u) => <span className="text-[13px] tabular text-ink-500">{formatDateTime(u.last_login_at)}</span>,
    },
    {
      key: 'created',
      header: 'Joined',
      sortKey: 'created_at',
      cell: (u) => <span className="text-[13px] tabular text-ink-500">{formatDateTime(u.created_at)}</span>,
    },
    {
      key: 'actions',
      header: '',
      className: 'w-24 text-right',
      cell: (u) => (
        <div className="flex justify-end gap-1">
          <button
            onClick={(e) => { e.stopPropagation(); setEditing(u); setFormOpen(true); }}
            aria-label={`Edit ${u.name}`}
            className="rounded-lg p-2 text-ink-400 transition-colors hover:bg-primary-50 hover:text-primary-600 cursor-pointer"
          >
            <Pencil className="size-4" />
          </button>
          <button
            onClick={(e) => { e.stopPropagation(); setDeleting(u); }}
            aria-label={`Delete ${u.name}`}
            className="rounded-lg p-2 text-ink-400 transition-colors hover:bg-danger-50 hover:text-danger-600 cursor-pointer"
          >
            <Trash2 className="size-4" />
          </button>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-5 animate-fade-up">
      <PageHeader
        title="Users"
        description="Manage accounts, roles, and access."
        actions={
          <Button
            icon={<Plus className="size-4" />}
            onClick={() => { setEditing(null); setFormOpen(true); }}
          >
            New user
          </Button>
        }
      />

      {/* Toolbar */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1 sm:max-w-sm">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-300" aria-hidden />
          <input
            value={table.search}
            onChange={(e) => table.setSearch(e.target.value)}
            placeholder="Search by name or email…"
            className="h-9.5 w-full rounded-lg border border-ink-200 bg-white pl-9 pr-3 text-sm placeholder:text-ink-300 transition-all focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-500/30"
          />
        </div>
        <div className="w-full sm:w-44">
          <Select
            aria-label="Filter by status"
            value={String(table.filters.status ?? '')}
            onChange={(e) => table.setFilter('status', e.target.value)}
          >
            <option value="">All statuses</option>
            <option value="active">Active</option>
            <option value="locked">Locked</option>
            <option value="pending">Pending</option>
          </Select>
        </div>
        {table.isFetching && <span className="text-xs text-ink-400">Refreshing…</span>}
      </div>

      <DataTable<User>
        columns={columns}
        rows={table.rows}
        meta={table.meta}
        loading={table.isLoading}
        error={table.error}
        onRetry={() => void table.refetch()}
        sortBy={table.sortBy}
        sortDir={table.sortDir}
        onSort={table.toggleSort}
        rowKey={(u) => u.id}
        emptyTitle="No users found"
        emptyDescription="Try adjusting your search or filters, or create a new user."
        summary={`${table.summary} ${table.total === 1 ? 'user' : 'users'}`}
        footer={
          <PaginationBar
            meta={table.meta}
            onNext={table.next}
            onPrev={table.prev}
            canPrev={table.canPrev}
          />
        }
      />

      <UserFormModal open={formOpen} onClose={() => setFormOpen(false)} user={editing} />

      <ConfirmDialog
        open={Boolean(deleting)}
        onCancel={() => setDeleting(null)}
        onConfirm={() => deleting && deleteMutation.mutate(deleting.id)}
        loading={deleteMutation.isPending}
        title="Delete user"
        message={`This will permanently remove ${deleting?.name || deleting?.email}. Their documents and audit history remain.`}
        confirmLabel="Delete user"
      />
    </div>
  );
}
