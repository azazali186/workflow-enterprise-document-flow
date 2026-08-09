import { useEffect, useState, type FormEvent } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Modal } from '@/components/ui/Modal';
import { Button } from '@/components/ui/Button';
import { TextInput, Textarea } from '@/components/ui/Field';
import { templatesService } from '@/services/workflow.service';
import { useToast, errorMessage } from '@/hooks/useToast';
import type { Template } from '@/types/entities';

interface Props {
  open: boolean;
  onClose: () => void;
  template: Template | null;
}

export function TemplateFormModal({ open, onClose, template }: Props) {
  const qc = useQueryClient();
  const toast = useToast();
  const editing = Boolean(template);

  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [description, setDescription] = useState('');
  const [content, setContent] = useState('');

  useEffect(() => {
    if (open) {
      setName(template?.name ?? '');
      setSlug(template?.slug ?? '');
      setDescription(template?.description ?? '');
      setContent(template?.content ?? '');
    }
  }, [open, template]);

  const mutation = useMutation({
    mutationFn: () =>
      editing && template
        ? templatesService.update({ id: template.id, name, description, content })
        : templatesService.create({ name, slug, description, content }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['templates'] });
      toast.success(editing ? 'Template updated' : 'Template created');
      onClose();
    },
    onError: (err) => toast.error('Could not save template', errorMessage(err)),
  });

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (!name.trim() || (!editing && !slug.trim())) return;
    mutation.mutate();
  };

  return (
    <Modal open={open} onClose={onClose} title={editing ? 'Edit template' : 'New template'} size="md">
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        <TextInput label="Name" required value={name} onChange={(e) => setName(e.target.value)} placeholder="Purchase order" />
        {!editing && (
          <TextInput label="Slug" required value={slug} onChange={(e) => setSlug(e.target.value)} placeholder="purchase-order" />
        )}
        <TextInput label="Description" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Optional summary" />
        <Textarea label="Content" value={content} onChange={(e) => setContent(e.target.value)} rows={6} placeholder="Template body…" />
        <div className="flex justify-end gap-2 pt-1">
          <Button type="button" variant="outline" onClick={onClose} disabled={mutation.isPending}>Cancel</Button>
          <Button type="submit" loading={mutation.isPending}>{editing ? 'Save changes' : 'Create template'}</Button>
        </div>
      </form>
    </Modal>
  );
}
