import { PageHeader } from '@/components/ui/Card';
import { Badge, statusTone } from '@/components/ui/Badge';
import { DataTable, PaginationBar, type Column } from '@/components/ui/DataTable';
import { usePaginatedQuery } from '@/hooks/usePaginatedQuery';
import { storagesService } from '@/services/workflow.service';
import { formatBytes, formatDateTime } from '@/lib/format';
import type { StorageRecord } from '@/types/entities';

export function StoragesPage() {
  const table = usePaginatedQuery<StorageRecord>(['storages'], (req, signal) => storagesService.list(req, signal), {
    defaultSortBy: 'created_at',
  });

  const columns: Column<StorageRecord>[] = [
    {
      key: 'document',
      header: 'Document',
      cell: (s) => (
        <div>
          <p className="font-medium text-ink-900">Document #{s.document_id.slice(0, 8)}</p>
          <p className="text-xs text-ink-400">{s.file_name || 'no file name'}</p>
        </div>
      ),
    },
    {
      key: 'provider',
      header: 'Provider',
      cell: (s) => <Badge tone="neutral">{s.provider}</Badge>,
    },
    {
      key: 'mime',
      header: 'Type',
      cell: (s) => <span className="text-[13px] text-ink-500">{s.mime_type || '—'}</span>,
    },
    {
      key: 'size',
      header: 'Size',
      sortKey: 'size_bytes',
      cell: (s) => <span className="text-[13px] tabular text-ink-500">{formatBytes(s.size_bytes)}</span>,
    },
    {
      key: 'status',
      header: 'Status',
      cell: (s) => <Badge tone={statusTone(s.status)} dot>{s.status}</Badge>,
    },
    {
      key: 'stored_at',
      header: 'Stored',
      cell: (s) => <span className="text-[13px] tabular text-ink-500">{formatDateTime(s.stored_at)}</span>,
    },
  ];

  return (
    <div className="space-y-5 animate-fade-up">
      <PageHeader title="Storages" description="Files attached to documents and where they live." />

      <DataTable<StorageRecord>
        columns={columns}
        rows={table.rows}
        meta={table.meta}
        loading={table.isLoading}
        error={table.error}
        onRetry={() => void table.refetch()}
        sortBy={table.sortBy}
        sortDir={table.sortDir}
        onSort={table.toggleSort}
        rowKey={(s) => s.id}
        emptyTitle="No storage records"
        emptyDescription="Uploaded files appear here after a document gets one."
        summary={table.summary}
        footer={<PaginationBar meta={table.meta} onNext={table.next} onPrev={table.prev} canPrev={table.canPrev} />}
      />
    </div>
  );
}
