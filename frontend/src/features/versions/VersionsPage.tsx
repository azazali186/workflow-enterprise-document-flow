import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { PageHeader } from '@/components/ui/Card';
import { TextInput } from '@/components/ui/Field';
import { Badge } from '@/components/ui/Badge';
import { DataTable, PaginationBar, type Column } from '@/components/ui/DataTable';
import { useCursorPagination } from '@/hooks/useCursorPagination';
import { versionsService } from '@/services/workflow.service';
import { formatDateTime } from '@/lib/format';
import type { Version } from '@/types/entities';

export function VersionsPage() {
  const [documentId, setDocumentId] = useState('');
  const pagination = useCursorPagination(20);

  const query = useQuery({
    queryKey: ['versions', documentId, pagination.cursor, pagination.limit],
    queryFn: ({ signal }) => versionsService.list(documentId, { cursor: pagination.cursor || undefined, limit: pagination.limit }, signal),
    enabled: documentId.trim().length > 0,
    placeholderData: (prev) => prev,
  });

  const columns: Column<Version>[] = [
    {
      key: 'version',
      header: 'Version',
      sortKey: 'version_number',
      cell: (v) => (
        <div className="flex items-center gap-2">
          <span className="font-medium text-ink-900">v{v.version_number}</span>
        </div>
      ),
    },
    {
      key: 'summary',
      header: 'Change summary',
      cell: (v) => <span className="max-w-72 truncate text-[13px] text-ink-500">{v.change_summary || '—'}</span>,
    },
    {
      key: 'created_by',
      header: 'Created by',
      cell: (v) => <span className="text-[13px] text-ink-500">{v.created_by ? `#${v.created_by.slice(0, 8)}` : '—'}</span>,
    },
    {
      key: 'created_at',
      header: 'Created',
      sortKey: 'created_at',
      cell: (v) => <span className="text-[13px] tabular text-ink-500">{formatDateTime(v.created_at)}</span>,
    },
  ];

  return (
    <div className="space-y-5 animate-fade-up">
      <PageHeader title="Versions" description="Snapshot history for one document." />

      <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
        <div className="flex-1 sm:max-w-md">
          <TextInput
            label="Document ID"
            value={documentId}
            onChange={(e) => { setDocumentId(e.target.value); pagination.reset(); }}
            placeholder="Paste a document UUID to load its versions"
          />
        </div>
      </div>

      {!documentId.trim() ? (
        <div className="rounded-xl border border-dashed border-ink-200 bg-white/60 px-6 py-14 text-center">
          <p className="text-sm font-medium text-ink-700">Enter a document ID to see its version history</p>
          <p className="mt-1 text-[13px] text-ink-400">Find document IDs on the Documents page.</p>
        </div>
      ) : query.isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="h-12 animate-pulse rounded-xl bg-ink-100" />
          ))}
        </div>
      ) : (
        <DataTable<Version>
          columns={columns}
          rows={query.data?.items}
          meta={query.data?.pagination}
          loading={query.isLoading}
          error={query.error}
          onRetry={() => void query.refetch()}
          sortBy="version_number"
          sortDir="desc"
          onSort={() => undefined}
          rowKey={(v) => v.id}
          emptyTitle="No versions"
          emptyDescription="This document has no version history yet."
          summary={pagination.summary}
          footer={
            <PaginationBar
              meta={query.data?.pagination}
              onNext={() => pagination.next(query.data?.pagination.next_cursor)}
              onPrev={pagination.prev}
              canPrev={pagination.canPrev}
            />
          }
        />
      )}

      {query.data?.items && query.data.items.length > 0 && (
        <Badge tone="neutral">Latest snapshot: v{Math.max(...query.data.items.map((v) => v.version_number))}</Badge>
      )}
    </div>
  );
}
