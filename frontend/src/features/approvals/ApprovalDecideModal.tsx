import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Modal } from '@/components/ui/Modal';
import { Button } from '@/components/ui/Button';
import { Textarea } from '@/components/ui/Field';
import { approvalsService } from '@/services/workflow.service';
import { useToast, errorMessage } from '@/hooks/useToast';
import type { Approval } from '@/types/entities';

interface Props {
  approval: Approval | null;
  onClose: () => void;
}

export function ApprovalDecideModal({ approval, onClose }: Props) {
  const qc = useQueryClient();
  const toast = useToast();
  const [comment, setComment] = useState('');

  const mutation = useMutation({
    mutationFn: (decision: 'approved' | 'rejected') => {
      if (!approval) return Promise.reject(new Error('No approval selected'));
      return approvalsService.decide({ approval_id: approval.id, decision, comment });
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['approvals'] });
      toast.success('Decision recorded');
      onClose();
    },
    onError: (err) => toast.error('Could not record decision', errorMessage(err)),
  });

  return (
    <Modal
      open={Boolean(approval)}
      onClose={onClose}
      title="Decide approval"
      description={
        approval
          ? `Level ${approval.level} · Document ${approval.document_id.slice(0, 8)} — currently ${approval.status}`
          : undefined
      }
      footer={
        <>
          <span className="mr-auto text-[13px] text-ink-500">Decision is final for this step.</span>
          <Button variant="outline" onClick={onClose} disabled={mutation.isPending}>Cancel</Button>
          <Button
            variant="danger"
            onClick={() => mutation.mutate('rejected')}
            loading={mutation.isPending && mutation.variables === 'rejected'}
          >
            Reject
          </Button>
          <Button
            onClick={() => mutation.mutate('approved')}
            loading={mutation.isPending && mutation.variables === 'approved'}
          >
            Approve
          </Button>
        </>
      }
    >
      <Textarea
        label="Comment (optional)"
        placeholder="Add context for the decision…"
        value={comment}
        onChange={(e) => setComment(e.target.value)}
        rows={4}
      />
    </Modal>
  );
}
