import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { FolderTree, Pencil, Plus, Trash2 } from 'lucide-react';
import { PageHeader } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { DataTable, PaginationBar, type Column } from '@/components/ui/DataTable';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { CategoryFormModal } from './CategoryFormModal';
import { usePaginatedQuery } from '@/hooks/usePaginatedQuery';
import { useToast, errorMessage } from '@/hooks/useToast';
import { categoriesService } from '@/services/categories.service';
import type { Category } from '@/types/entities';

export function CategoriesPage() {
  const qc = useQueryClient();
  const toast = useToast();
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Category | null>(null);
  const [deleting, setDeleting] = useState<Category | null>(null);

  const table = usePaginatedQuery<Category>(['categories'], (req, signal) => categoriesService.list(req, signal), {
    defaultSortBy: 'sort_order',
    defaultSortDir: 'asc',
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => categoriesService.remove(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['categories'] });
      toast.success('Category deleted');
      setDeleting(null);
    },
    onError: (err) => toast.error('Could not delete category', errorMessage(err)),
  });

  const columns: Column<Category>[] = [
    {
      key: 'name',
      header: 'Category',
      sortKey: 'name',
      cell: (c) => (
        <div className="flex items-center gap-3">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-paper-100 text-ink-400">
            <FolderTree className="size-4.5" aria-hidden />
          </div>
          <div>
            <p className="font-medium text-ink-900">{c.name}</p>
            <p className="text-xs text-ink-400">/{c.slug}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'description',
      header: 'Description',
      cell: (c) => <span className="text-[13px] text-ink-500">{c.description || '—'}</span>,
    },
    {
      key: 'sort',
      header: 'Order',
      sortKey: 'sort_order',
      cell: (c) => <span className="tabular text-[13px] text-ink-500">{c.sort_order}</span>,
    },
    {
      key: 'active',
      header: 'Status',
      cell: (c) => (c.is_active ? <Badge tone="success" dot>Active</Badge> : <Badge tone="neutral" dot>Inactive</Badge>),
    },
    {
      key: 'actions',
      header: '',
      className: 'w-24 text-right',
      cell: (c) => (
        <div className="flex justify-end gap-1">
          <button
            onClick={(e) => { e.stopPropagation(); setEditing(c); setFormOpen(true); }}
            aria-label={`Edit ${c.name}`}
            className="rounded-lg p-2 text-ink-400 transition-colors hover:bg-primary-50 hover:text-primary-600 cursor-pointer"
          >
            <Pencil className="size-4" />
          </button>
          <button
            onClick={(e) => { e.stopPropagation(); setDeleting(c); }}
            aria-label={`Delete ${c.name}`}
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
        title="Categories"
        description="Organize documents into a clean taxonomy."
        actions={
          <Button icon={<Plus className="size-4" />} onClick={() => { setEditing(null); setFormOpen(true); }}>
            New category
          </Button>
        }
      />

      <DataTable<Category>
        columns={columns}
        rows={table.rows}
        meta={table.meta}
        loading={table.isLoading}
        error={table.error}
        onRetry={() => void table.refetch()}
        sortBy={table.sortBy}
        sortDir={table.sortDir}
        onSort={table.toggleSort}
        rowKey={(c) => c.id}
        emptyTitle="No categories yet"
        emptyDescription="Create categories to keep documents organized."
        summary={table.summary}
        footer={<PaginationBar meta={table.meta} onNext={table.next} onPrev={table.prev} canPrev={table.canPrev} />}
      />

      <CategoryFormModal open={formOpen} onClose={() => setFormOpen(false)} category={editing} />

      <ConfirmDialog
        open={Boolean(deleting)}
        onCancel={() => setDeleting(null)}
        onConfirm={() => deleting && deleteMutation.mutate(deleting.id)}
        loading={deleteMutation.isPending}
        title="Delete category"
        message={`Delete "${deleting?.name}"? Documents in it will become uncategorized.`}
        confirmLabel="Delete category"
      />
    </div>
  );
}
