import { useEffect, useState, type FormEvent } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Modal } from '@/components/ui/Modal';
import { Button } from '@/components/ui/Button';
import { TextInput, Textarea } from '@/components/ui/Field';
import { categoriesService } from '@/services/categories.service';
import { useToast, errorMessage } from '@/hooks/useToast';
import type { Category } from '@/types/entities';

interface CategoryFormModalProps {
  open: boolean;
  onClose: () => void;
  category?: Category | null;
}

export function CategoryFormModal({ open, onClose, category }: CategoryFormModalProps) {
  const qc = useQueryClient();
  const toast = useToast();
  const isEdit = Boolean(category);

  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [description, setDescription] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!open) return;
    setName(category?.name ?? '');
    setSlug(category?.slug ?? '');
    setDescription(category?.description ?? '');
    setErrors({});
  }, [open, category]);

  const mutation = useMutation({
    mutationFn: () =>
      isEdit
        ? categoriesService.update({ id: category!.id, name, description })
        : categoriesService.create({ name, slug, description }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['categories'] });
      toast.success(isEdit ? 'Category updated' : 'Category created');
      onClose();
    },
    onError: (err) => toast.error('Could not save category', errorMessage(err)),
  });

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    const errs: Record<string, string> = {};
    if (!name.trim()) errs.name = 'Name is required.';
    if (!isEdit && !/^[a-z0-9-]+$/.test(slug.trim())) errs.slug = 'Use lowercase letters, numbers, and hyphens.';
    setErrors(errs);
    if (Object.keys(errs).length) return;
    mutation.mutate();
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit category' : 'New category'}
      footer={
        <>
          <Button variant="outline" onClick={onClose} disabled={mutation.isPending}>Cancel</Button>
          <Button type="submit" form="category-form" loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Create category'}
          </Button>
        </>
      }
    >
      <form id="category-form" onSubmit={onSubmit} className="space-y-4" noValidate>
        <TextInput label="Name" value={name} onChange={(e) => setName(e.target.value)} error={errors.name} placeholder="Financial Reports" required />
        {!isEdit && (
          <TextInput
            label="Slug"
            value={slug}
            onChange={(e) => setSlug(e.target.value)}
            error={errors.slug}
            placeholder="financial-reports"
            hint="Used in URLs and as a unique key."
            required
          />
        )}
        <Textarea label="Description" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="What belongs in this category?" />
      </form>
    </Modal>
  );
}
