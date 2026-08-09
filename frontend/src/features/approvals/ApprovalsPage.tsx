import { useState } from 'react';
import { CheckCircle2 } from 'lucide-react';
import { PageHeader } from '@/components/ui/Card';
import { Select } from '@/components/ui/Field';
import { Badge, statusTone } from '@/components/ui/Badge';
import { DataTable, PaginationBar, type Column } from '@/components/ui/DataTable';
import { usePaginatedQuery } from '@/hooks/usePaginatedQuery';
import { approvalsService } from '@/services/workflow.service';
import { formatDateTime } from '@/lib/format';
import { ApprovalDecideModal } from './ApprovalDecideModal';
import type { Approval } from '@/types/entities';

export function ApprovalsPage() {
  const [deciding, setDeciding] = useState<Approval | null>(null);

  const table = usePaginatedQuery<Approval>(['approvals'], (req, signal) => approvalsService.list(req, signal), {
    defaultSortBy: 'created_at',
  });

  const columns: Column<Approval>[] = [
    {
      key: 'document',
      header: 'Document',
      cell: (a) => (
        <div>
          <p className="font-medium text-ink-900">Document #{a.document_id.slice(0, 8)}</p>
          <p className="text-xs text-ink-400">Level {a.level}</p>
        </div>
      ),
    },
    {
      key: 'approver',
      header: 'Approver',
      cell: (a) => <span className="text-[13px] text-ink-500">#{a.approver_id.slice(0, 8)}</span>,
    },
    {
      key: 'status',
      header: 'Status',
      cell: (a) => <Badge tone={statusTone(a.status)} dot>{a.status}</Badge>,
    },
    {
      key: 'comment',
      header: 'Comment',
      cell: (a) => <span className="max-w-56 truncate text-[13px] text-ink-500">{a.comment || '—'}</span>,
    },
    {
      key: 'decided',
      header: 'Decided',
      cell: (a) => <span className="text-[13px] tabular text-ink-500">{formatDateTime(a.decided_at)}</span>,
    },
    {
      key: 'actions',
      header: '',
      className: 'w-24 text-right',
      cell: (a) =>
        a.status === 'pending' ? (
          <button
            onClick={(e) => { e.stopPropagation(); setDeciding(a); }}
            title="Approve or reject"
            aria-label={`Decide approval for document ${a.document_id.slice(0, 8)}`}
            className="inline-flex items-center gap-1.5 rounded-lg border border-primary-200 bg-primary-50 px-2.5 py-1.5 text-xs font-medium text-primary-700 transition-colors hover:bg-primary-100 cursor-pointer"
          >
            <CheckCircle2 className="size-3.5" />
            Decide
          </button>
        ) : (
          <span className="block text-right text-xs text-ink-300">Closed</span>
        ),
    },
  ];

  return (
    <div className="space-y-5 animate-fade-up">
      <PageHeader
        title="Approvals"
        description="Review and decide multi-level document approvals."
        actions={
          <div className="w-44">
            <Select
              aria-label="Filter by status"
              value={String(table.filters.status ?? '')}
              onChange={(e) => table.setFilter('status', e.target.value)}
            >
              <option value="">All statuses</option>
              <option value="pending">Pending</option>
              <option value="approved">Approved</option>
              <option value="rejected">Rejected</option>
            </Select>
          </div>
        }
      />

      <DataTable<Approval>
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
        emptyTitle="No approvals yet"
        emptyDescription="Approval chains appear here once they are opened on a document."
        summary={table.summary}
        footer={<PaginationBar meta={table.meta} onNext={table.next} onPrev={table.prev} canPrev={table.canPrev} />}
      />

      <ApprovalDecideModal approval={deciding} onClose={() => setDeciding(null)} />
    </div>
  );
}
