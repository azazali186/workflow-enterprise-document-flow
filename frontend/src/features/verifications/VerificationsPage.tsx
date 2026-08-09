import { useState } from 'react';
import { ShieldCheck } from 'lucide-react';
import { PageHeader } from '@/components/ui/Card';
import { Select } from '@/components/ui/Field';
import { Badge, statusTone } from '@/components/ui/Badge';
import { DataTable, PaginationBar, type Column } from '@/components/ui/DataTable';
import { usePaginatedQuery } from '@/hooks/usePaginatedQuery';
import { verificationsService } from '@/services/workflow.service';
import { formatDateTime } from '@/lib/format';
import { VerificationDecideModal } from './VerificationDecideModal';
import type { Verification } from '@/types/entities';

export function VerificationsPage() {
  const [deciding, setDeciding] = useState<Verification | null>(null);

  const table = usePaginatedQuery<Verification>(['verifications'], (req, signal) => verificationsService.list(req, signal), {
    defaultSortBy: 'created_at',
  });

  const columns: Column<Verification>[] = [
    {
      key: 'document',
      header: 'Document',
      cell: (v) => (
        <div>
          <p className="font-medium text-ink-900">Document #{v.document_id.slice(0, 8)}</p>
          <p className="text-xs text-ink-400">Requested by #{v.requested_by?.slice(0, 8) || '—'}</p>
        </div>
      ),
    },
    {
      key: 'method',
      header: 'Method',
      cell: (v) => <Badge tone="neutral">{v.method || 'manual'}</Badge>,
    },
    {
      key: 'status',
      header: 'Status',
      cell: (v) => <Badge tone={statusTone(v.status)} dot>{v.status}</Badge>,
    },
    {
      key: 'verified_by',
      header: 'Verified by',
      cell: (v) => <span className="text-[13px] text-ink-500">{v.verified_by ? `#${v.verified_by.slice(0, 8)}` : '—'}</span>,
    },
    {
      key: 'verified_at',
      header: 'Verified at',
      cell: (v) => <span className="text-[13px] tabular text-ink-500">{formatDateTime(v.verified_at)}</span>,
    },
    {
      key: 'actions',
      header: '',
      className: 'w-24 text-right',
      cell: (v) =>
        v.status === 'pending' ? (
          <button
            onClick={(e) => { e.stopPropagation(); setDeciding(v); }}
            title="Verify or reject"
            aria-label={`Decide verification for document ${v.document_id.slice(0, 8)}`}
            className="inline-flex items-center gap-1.5 rounded-lg border border-primary-200 bg-primary-50 px-2.5 py-1.5 text-xs font-medium text-primary-700 transition-colors hover:bg-primary-100 cursor-pointer"
          >
            <ShieldCheck className="size-3.5" />
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
        title="Verifications"
        description="Confirm document authenticity before approval."
        actions={
          <div className="w-44">
            <Select
              aria-label="Filter by status"
              value={String(table.filters.status ?? '')}
              onChange={(e) => table.setFilter('status', e.target.value)}
            >
              <option value="">All statuses</option>
              <option value="pending">Pending</option>
              <option value="verified">Verified</option>
              <option value="rejected">Rejected</option>
            </Select>
          </div>
        }
      />

      <DataTable<Verification>
        columns={columns}
        rows={table.rows}
        meta={table.meta}
        loading={table.isLoading}
        error={table.error}
        onRetry={() => void table.refetch()}
        sortBy={table.sortBy}
        sortDir={table.sortDir}
        onSort={table.toggleSort}
        rowKey={(v) => v.id}
        emptyTitle="No verifications yet"
        emptyDescription="Verification requests appear here once they are opened on a document."
        summary={table.summary}
        footer={<PaginationBar meta={table.meta} onNext={table.next} onPrev={table.prev} canPrev={table.canPrev} />}
      />

      <VerificationDecideModal verification={deciding} onClose={() => setDeciding(null)} />
    </div>
  );
}
