import { useEffect, useState, type FormEvent } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Modal } from '@/components/ui/Modal';
import { Button } from '@/components/ui/Button';
import { TextInput, Textarea } from '@/components/ui/Field';
import { rolesService } from '@/services/roles.service';
import { useToast, errorMessage } from '@/hooks/useToast';
import type { Role } from '@/types/entities';

interface RoleFormModalProps {
  open: boolean;
  onClose: () => void;
  role?: Role | null;
}

export function RoleFormModal({ open, onClose, role }: RoleFormModalProps) {
  const qc = useQueryClient();
  const toast = useToast();
  const isEdit = Boolean(role);

  const [code, setCode] = useState('');
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!open) return;
    setCode(role?.code ?? '');
    setName(role?.name ?? '');
    setDescription(role?.description ?? '');
    setErrors({});
  }, [open, role]);

  const mutation = useMutation({
    mutationFn: () =>
      isEdit
        ? rolesService.update({ id: role!.id, name, description })
        : rolesService.create({ code, name, description }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['roles'] });
      toast.success(isEdit ? 'Role updated' : 'Role created');
      onClose();
    },
    onError: (err) => toast.error('Could not save role', errorMessage(err)),
  });

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    const errs: Record<string, string> = {};
    if (!name.trim()) errs.name = 'Name is required.';
    if (!isEdit && !/^[a-z0-9_]+$/.test(code.trim())) errs.code = 'Use lowercase letters, numbers, and underscores.';
    setErrors(errs);
    if (Object.keys(errs).length) return;
    mutation.mutate();
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit role' : 'Create role'}
      footer={
        <>
          <Button variant="outline" onClick={onClose} disabled={mutation.isPending}>Cancel</Button>
          <Button type="submit" form="role-form" loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Create role'}
          </Button>
        </>
      }
    >
      <form id="role-form" onSubmit={onSubmit} className="space-y-4" noValidate>
        {!isEdit && (
          <TextInput
            label="Role code"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            error={errors.code}
            placeholder="finance_manager"
            hint="A unique identifier used by the system."
            required
          />
        )}
        <TextInput label="Display name" value={name} onChange={(e) => setName(e.target.value)} error={errors.name} placeholder="Finance Manager" required />
        <Textarea label="Description" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="What can this role do?" />
      </form>
    </Modal>
  );
}
