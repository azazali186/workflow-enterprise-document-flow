import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Pencil, Plus, Trash2 } from 'lucide-react';
import { PageHeader } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { DataTable, PaginationBar, type Column } from '@/components/ui/DataTable';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { usePaginatedQuery } from '@/hooks/usePaginatedQuery';
import { useToast, errorMessage } from '@/hooks/useToast';
import { templatesService } from '@/services/workflow.service';
import { TemplateFormModal } from './TemplateFormModal';
import type { Template } from '@/types/entities';

export function TemplatesPage() {
  const qc = useQueryClient();
  const toast = useToast();
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Template | null>(null);
  const [deleting, setDeleting] = useState<Template | null>(null);

  const table = usePaginatedQuery<Template>(['templates'], (req, signal) => templatesService.list(req, signal), {
    defaultSortBy: 'name',
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => templatesService.remove(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['templates'] });
      toast.success('Template deleted');
      setDeleting(null);
    },
    onError: (err) => toast.error('Could not delete template', errorMessage(err)),
  });

  const columns: Column<Template>[] = [
    {
      key: 'name',
      header: 'Template',
      sortKey: 'name',
      cell: (t) => (
        <div>
          <p className="font-medium text-ink-900">{t.name}</p>
          <p className="text-xs text-ink-400">/{t.slug}</p>
        </div>
      ),
    },
    {
      key: 'description',
      header: 'Description',
      cell: (t) => <span className="max-w-64 truncate text-[13px] text-ink-500">{t.description || '—'}</span>,
    },
    {
      key: 'version',
      header: 'Version',
      cell: (t) => <span className="text-[13px] tabular text-ink-500">v{t.version}</span>,
    },
    {
      key: 'is_active',
      header: 'Status',
      cell: (t) => (t.is_active ? <Badge tone="success">Active</Badge> : <Badge tone="neutral">Inactive</Badge>),
    },
    {
      key: 'actions',
      header: '',
      className: 'w-24 text-right',
      cell: (t) => (
        <div className="flex justify-end gap-1">
          <button
            onClick={(e) => { e.stopPropagation(); setEditing(t); setFormOpen(true); }}
            aria-label={`Edit ${t.name}`}
            className="rounded-lg p-2 text-ink-400 transition-colors hover:bg-primary-50 hover:text-primary-600 cursor-pointer"
          >
            <Pencil className="size-4" />
          </button>
          <button
            onClick={(e) => { e.stopPropagation(); setDeleting(t); }}
            aria-label={`Delete ${t.name}`}
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
        title="Templates"
        description="Reusable document scaffolds with versioned content."
        actions={
          <Button icon={<Plus className="size-4" />} onClick={() => { setEditing(null); setFormOpen(true); }}>
            New template
          </Button>
        }
      />

      <DataTable<Template>
        columns={columns}
        rows={table.rows}
        meta={table.meta}
        loading={table.isLoading}
        error={table.error}
        onRetry={() => void table.refetch()}
        sortBy={table.sortBy}
        sortDir={table.sortDir}
        onSort={table.toggleSort}
        rowKey={(t) => t.id}
        emptyTitle="No templates yet"
        emptyDescription="Create a template to speed up document creation."
        summary={table.summary}
        footer={<PaginationBar meta={table.meta} onNext={table.next} onPrev={table.prev} canPrev={table.canPrev} />}
      />

      <TemplateFormModal open={formOpen} onClose={() => setFormOpen(false)} template={editing} />

      <ConfirmDialog
        open={Boolean(deleting)}
        onCancel={() => setDeleting(null)}
        onConfirm={() => deleting && deleteMutation.mutate(deleting.id)}
        loading={deleteMutation.isPending}
        title="Delete template"
        message={`Delete "${deleting?.name}"? This cannot be undone.`}
        confirmLabel="Delete template"
      />
    </div>
  );
}
