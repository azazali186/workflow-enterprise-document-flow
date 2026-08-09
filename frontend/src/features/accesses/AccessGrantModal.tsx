import { useState, type FormEvent } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Modal } from '@/components/ui/Modal';
import { Button } from '@/components/ui/Button';
import { TextInput, Select } from '@/components/ui/Field';
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
        <TextInput label="Document ID" required value={documentId} onChange={(e) => setDocumentId(e.target.value)} placeholder="UUID of the document" />
        <TextInput label="User ID (or leave blank for role-wide)" value={userId} onChange={(e) => setUserId(e.target.value)} placeholder="UUID of the user" />
        <TextInput label="Role ID (or leave blank for one user)" value={roleId} onChange={(e) => setRoleId(e.target.value)} placeholder="UUID of the role" />
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
