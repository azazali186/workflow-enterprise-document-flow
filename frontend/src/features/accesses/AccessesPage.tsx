import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Undo2 } from 'lucide-react';
import { PageHeader } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { DataTable, PaginationBar, type Column } from '@/components/ui/DataTable';
import { usePaginatedQuery } from '@/hooks/usePaginatedQuery';
import { useToast, errorMessage } from '@/hooks/useToast';
import { accessesService } from '@/services/workflow.service';
import { formatDateTime } from '@/lib/format';
import { AccessGrantModal } from './AccessGrantModal';
import type { Access } from '@/types/entities';

const permissionTone: Record<Access['permission'], 'primary' | 'success' | 'warning'> = {
  read: 'primary',
  write: 'success',
  approve: 'warning',
};

export function AccessesPage() {
  const qc = useQueryClient();
  const toast = useToast();
  const [grantOpen, setGrantOpen] = useState(false);

  const table = usePaginatedQuery<Access>(['accesses'], (req, signal) => accessesService.list(req, signal), {
    defaultSortBy: 'created_at',
  });

  const revokeMutation = useMutation({
    mutationFn: (id: string) => accessesService.revoke(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['accesses'] });
      toast.success('Access revoked');
    },
    onError: (err) => toast.error('Could not revoke access', errorMessage(err)),
  });

  const columns: Column<Access>[] = [
    {
      key: 'document',
      header: 'Document',
      cell: (a) => <span className="font-medium text-ink-900">Document #{a.document_id.slice(0, 8)}</span>,
    },
    {
      key: 'grantee',
      header: 'Grantee',
      cell: (a) => (
        <div>
          {a.user_id ? (
            <>
              <p className="text-[13px] text-ink-700">User #{a.user_id.slice(0, 8)}</p>
              <p className="text-xs text-ink-400">direct</p>
            </>
          ) : a.role_id ? (
            <>
              <p className="text-[13px] text-ink-700">Role #{a.role_id.slice(0, 8)}</p>
              <p className="text-xs text-ink-400">role-wide</p>
            </>
          ) : (
            <span className="text-xs text-ink-300">—</span>
          )}
        </div>
      ),
    },
    {
      key: 'permission',
      header: 'Permission',
      cell: (a) => <Badge tone={permissionTone[a.permission]}>{a.permission}</Badge>,
    },
    {
      key: 'status',
      header: 'Status',
      cell: (a) =>
        a.revoked_at ? <Badge tone="neutral">Revoked</Badge> : <Badge tone="success">Active</Badge>,
    },
    {
      key: 'granted_at',
      header: 'Granted',
      cell: (a) => <span className="text-[13px] tabular text-ink-500">{formatDateTime(a.created_at)}</span>,
    },
    {
      key: 'actions',
      header: '',
      className: 'w-16 text-right',
      cell: (a) =>
        !a.revoked_at ? (
          <button
            onClick={(e) => { e.stopPropagation(); revokeMutation.mutate(a.id); }}
            disabled={revokeMutation.isPending}
            title="Revoke access"
            aria-label={`Revoke access for document ${a.document_id.slice(0, 8)}`}
            className="rounded-lg p-2 text-ink-400 transition-colors hover:bg-danger-50 hover:text-danger-600 disabled:opacity-40 cursor-pointer"
          >
            <Undo2 className="size-4" />
          </button>
        ) : null,
    },
  ];

  return (
    <div className="space-y-5 animate-fade-up">
      <PageHeader
        title="Access grants"
        description="Explicit per-document grants to users and roles."
        actions={
          <Button icon={<Plus className="size-4" />} onClick={() => setGrantOpen(true)}>
            Grant access
          </Button>
        }
      />

      <DataTable<Access>
        columns={columns}
        rows={table.rows}
        meta={table.meta}
        loading={table.isLoading}
        error={table.error}
        onRetry={() => void table.refetch()}
        sortBy={table.sortBy}
        sortDir={table.sortDir}
        onSort={table.toggleSort}
        rowKey={(a) => a.id}
        emptyTitle="No access grants"
        emptyDescription="Document owners grant access through the API; grants appear here."
        summary={table.summary}
        footer={<PaginationBar meta={table.meta} onNext={table.next} onPrev={table.prev} canPrev={table.canPrev} />}
      />

      <AccessGrantModal open={grantOpen} onClose={() => setGrantOpen(false)} />
    </div>
  );
}
