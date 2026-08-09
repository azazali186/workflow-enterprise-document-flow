import { useEffect, useState, type FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Modal } from '@/components/ui/Modal';
import { Button } from '@/components/ui/Button';
import { Select, TextInput, Textarea } from '@/components/ui/Field';
import { documentsService } from '@/services/documents.service';
import { categoriesService } from '@/services/categories.service';
import { useToast, errorMessage } from '@/hooks/useToast';

interface DocumentFormModalProps {
  open: boolean;
  onClose: () => void;
}

export function DocumentFormModal({ open, onClose }: DocumentFormModalProps) {
  const qc = useQueryClient();
  const toast = useToast();

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [categoryId, setCategoryId] = useState('');
  const [tagsInput, setTagsInput] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});

  const catsQuery = useQuery({
    queryKey: ['categories', 'all'],
    queryFn: () => categoriesService.list({ limit: 100, sort_by: 'sort_order', sort_dir: 'asc' }),
    enabled: open,
  });
  const categories = catsQuery.data?.items ?? [];

  useEffect(() => {
    if (!open) return;
    setTitle('');
    setDescription('');
    setCategoryId('');
    setTagsInput('');
    setErrors({});
  }, [open]);

  const mutation = useMutation({
    mutationFn: () =>
      documentsService.create({
        title,
        description: description || undefined,
        category_id: categoryId || undefined,
        tags: tagsInput
          .split(',')
          .map((t) => t.trim())
          .filter(Boolean),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['documents'] });
      toast.success('Document created', 'The upload pipeline has started.');
      onClose();
    },
    onError: (err) => toast.error('Could not create document', errorMessage(err)),
  });

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (!title.trim()) {
      setErrors({ title: 'Title is required.' });
      return;
    }
    mutation.mutate();
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="New document"
      description="Register a document — its workflow starts automatically."
      footer={
        <>
          <Button variant="outline" onClick={onClose} disabled={mutation.isPending}>Cancel</Button>
          <Button type="submit" form="document-form" loading={mutation.isPending}>Create document</Button>
        </>
      }
    >
      <form id="document-form" onSubmit={onSubmit} className="space-y-4" noValidate>
        <TextInput label="Title" value={title} onChange={(e) => setTitle(e.target.value)} error={errors.title} placeholder="Q3 Financial Report" required />
        <Textarea label="Description" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="A short summary of the document…" />
        <div className="grid gap-4 sm:grid-cols-2">
          <Select label="Category" value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
            <option value="">No category</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </Select>
          <TextInput label="Tags" value={tagsInput} onChange={(e) => setTagsInput(e.target.value)} placeholder="finance, q3, draft" hint="Comma separated" />
        </div>
      </form>
    </Modal>
  );
}
