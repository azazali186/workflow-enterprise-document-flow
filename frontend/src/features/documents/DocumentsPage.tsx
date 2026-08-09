import { useQuery } from '@tanstack/react-query';
import { FileText, Plus, Search } from 'lucide-react';
import { PageHeader } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Select } from '@/components/ui/Field';
import { SearchableSelect } from '@/components/ui/SearchableSelect';
import { Badge, statusTone } from '@/components/ui/Badge';
import { DataTable, PaginationBar, type Column } from '@/components/ui/DataTable';
import { DocumentFormModal } from './DocumentFormModal';
import { usePaginatedQuery } from '@/hooks/usePaginatedQuery';
import { categoriesService } from '@/services/categories.service';
import { documentsService } from '@/services/documents.service';
import { formatBytes, formatDateTime, humanize } from '@/lib/format';
import { useState } from 'react';
import type { Document } from '@/types/entities';

const DOC_STATUSES = ['draft', 'pending_verification', 'verified', 'rejected', 'approved', 'archived'];

export function DocumentsPage() {
  const [formOpen, setFormOpen] = useState(false);

  const table = usePaginatedQuery<Document>(['documents'], (req, signal) => documentsService.list(req, signal), {
    defaultSortBy: 'created_at',
  });

  const catsQuery = useQuery({
    queryKey: ['categories', 'all'],
    queryFn: () => categoriesService.list({ limit: 100, sort_by: 'sort_order', sort_dir: 'asc' }),
  });
  const categories = catsQuery.data?.items ?? [];
  const catName = (id?: string) => categories.find((c) => c.id === id)?.name;

  const columns: Column<Document>[] = [
    {
      key: 'title',
      header: 'Document',
      sortKey: 'title',
      cell: (d) => (
        <div className="flex items-center gap-3">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-paper-100 text-ink-400">
            <FileText className="size-4.5" aria-hidden />
          </div>
          <div className="min-w-0">
            <p className="truncate font-medium text-ink-900">{d.title}</p>
            <p className="text-xs tabular text-ink-400">{d.document_number}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      sortKey: 'status',
      cell: (d) => <Badge tone={statusTone(d.status)} dot>{humanize(d.status)}</Badge>,
    },
    {
      key: 'category',
      header: 'Category',
      cell: (d) => <span className="text-[13px] text-ink-500">{catName(d.category_id) || '—'}</span>,
    },
    {
      key: 'size',
      header: 'Size',
      sortKey: 'size_bytes',
      cell: (d) => <span className="text-[13px] tabular text-ink-500">{formatBytes(d.size_bytes)}</span>,
    },
    {
      key: 'tags',
      header: 'Tags',
      cell: (d) => (
        <div className="flex max-w-56 flex-wrap gap-1">
          {d.tags?.length ? (
            d.tags.slice(0, 3).map((t) => (
              <span key={t} className="rounded-md bg-ink-100 px-1.5 py-0.5 text-[11px] text-ink-500">{t}</span>
            ))
          ) : (
            <span className="text-xs text-ink-300">—</span>
          )}
        </div>
      ),
    },
    {
      key: 'created',
      header: 'Created',
      sortKey: 'created_at',
      cell: (d) => <span className="text-[13px] tabular text-ink-500">{formatDateTime(d.created_at)}</span>,
    },
  ];

  return (
    <div className="space-y-5 animate-fade-up">
      <PageHeader
        title="Documents"
        description="Create, track, and manage documents through their workflow."
        actions={
          <Button icon={<Plus className="size-4" />} onClick={() => setFormOpen(true)}>
            New document
          </Button>
        }
      />

      {/* Toolbar */}
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
        <div className="relative flex-1 lg:max-w-sm">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-300" aria-hidden />
          <input
            value={table.search}
            onChange={(e) => table.setSearch(e.target.value)}
            placeholder="Search documents…"
            className="h-9.5 w-full rounded-lg border border-ink-200 bg-white pl-9 pr-3 text-sm placeholder:text-ink-300 transition-all focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-500/30"
          />
        </div>
        <div className="grid grid-cols-2 gap-3 sm:flex sm:items-center">
          <Select
            aria-label="Filter by status"
            value={String(table.filters.status ?? '')}
            onChange={(e) => table.setFilter('status', e.target.value)}
          >
            <option value="">All statuses</option>
            {DOC_STATUSES.map((s) => (
              <option key={s} value={s}>{humanize(s)}</option>
            ))}
          </Select>
          <SearchableSelect
            kind="categories"
            ariaLabel="Filter by category"
            value={String(table.filters.category_id ?? '')}
            onChange={(id) => table.setFilter('category_id', id)}
            placeholder="All categories"
            allowClear
          />
        </div>
      </div>

      <DataTable<Document>
        columns={columns}
        rows={table.rows}
        meta={table.meta}
        loading={table.isLoading}
        error={table.error}
        onRetry={() => void table.refetch()}
        sortBy={table.sortBy}
        sortDir={table.sortDir}
        onSort={table.toggleSort}
        rowKey={(d) => d.id}
        emptyTitle="No documents found"
        emptyDescription="Adjust your filters or create your first document."
        summary={`${table.summary} ${table.total === 1 ? 'document' : 'documents'}`}
        footer={
          <PaginationBar
            meta={table.meta}
            onNext={table.next}
            onPrev={table.prev}
            canPrev={table.canPrev}
          />
        }
      />

      <DocumentFormModal open={formOpen} onClose={() => setFormOpen(false)} />
    </div>
  );
}
