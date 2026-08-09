import { useState, type FormEvent } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Modal } from '@/components/ui/Modal';
import { Button } from '@/components/ui/Button';
import { Select } from '@/components/ui/Field';
import { SearchableSelect } from '@/components/ui/SearchableSelect';
import { accessesService } from '@/services/workflow.service';
import { useToast, errorMessage } from '@/hooks/useToast';

interface Props {
  open: boolean;
  onClose: () => void;
}

export function AccessGrantModal({ open, onClose }: Props) {
  const qc = useQueryClient();
  const toast = useToast();
  const [documentId, setDocumentId] = useState('');
  const [userId, setUserId] = useState('');
  const [roleId, setRoleId] = useState('');
  const [permission, setPermission] = useState<'read' | 'write' | 'approve'>('read');

  const mutation = useMutation({
    mutationFn: () =>
      accessesService.grant({
        document_id: documentId,
        user_id: userId || undefined,
        role_id: roleId || undefined,
        permission,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['accesses'] });
      toast.success('Access granted');
      setDocumentId('');
      setUserId('');
      setRoleId('');
      setPermission('read');
      onClose();
    },
    onError: (err) => toast.error('Could not grant access', errorMessage(err)),
  });

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (!documentId.trim()) return;
    if (!userId.trim() && !roleId.trim()) return;
    mutation.mutate();
  };

  return (
    <Modal open={open} onClose={onClose} title="Grant access" size="md"
      description="Give a user or role explicit access to one document.">
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        <SearchableSelect
          label="Document"
          kind="documents"
          value={documentId}
          onChange={(id) => setDocumentId(id)}
          placeholder="Search documents…"
          required
          autoFocus
        />
        <SearchableSelect
          label="User"
          kind="users"
          value={userId}
          onChange={(id) => { setUserId(id); if (id) setRoleId(''); }}
          placeholder="Search users…"
          hint="Leave blank to grant via role instead"
          allowClear
        />
        <SearchableSelect
          label="Role"
          kind="roles"
          value={roleId}
          onChange={(id) => { setRoleId(id); if (id) setUserId(''); }}
          placeholder="Search roles…"
          hint="Leave blank to grant to one user"
          allowClear
        />
        <Select label="Permission" value={permission} onChange={(e) => setPermission(e.target.value as 'read' | 'write' | 'approve')}>
          <option value="read">Read</option>
          <option value="write">Write</option>
          <option value="approve">Approve</option>
        </Select>
        <div className="flex justify-end gap-2 pt-1">
          <Button type="button" variant="outline" onClick={onClose} disabled={mutation.isPending}>Cancel</Button>
          <Button type="submit" loading={mutation.isPending}>Grant access</Button>
        </div>
      </form>
    </Modal>
  );
}
