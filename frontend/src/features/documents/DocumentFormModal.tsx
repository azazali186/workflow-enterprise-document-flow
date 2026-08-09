import { useEffect, useState, type FormEvent } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Modal } from '@/components/ui/Modal';
import { Button } from '@/components/ui/Button';
import { TextInput, Textarea } from '@/components/ui/Field';
import { SearchableSelect } from '@/components/ui/SearchableSelect';
import { documentsService } from '@/services/documents.service';
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
          <SearchableSelect
            label="Category"
            kind="categories"
            value={categoryId}
            onChange={(id) => setCategoryId(id)}
            placeholder="Search categories…"
            hint="Optional — leave empty for uncategorised"
            allowClear
          />
          <TextInput label="Tags" value={tagsInput} onChange={(e) => setTagsInput(e.target.value)} placeholder="finance, q3, draft" hint="Comma separated" />
        </div>
      </form>
    </Modal>
  );
}
