import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Modal } from '@/components/ui/Modal';
import { Button } from '@/components/ui/Button';
import { Textarea } from '@/components/ui/Field';
import { verificationsService } from '@/services/workflow.service';
import { useToast, errorMessage } from '@/hooks/useToast';
import type { Verification } from '@/types/entities';

interface Props {
  verification: Verification | null;
  onClose: () => void;
}

export function VerificationDecideModal({ verification, onClose }: Props) {
  const qc = useQueryClient();
  const toast = useToast();
  const [notes, setNotes] = useState('');

  const mutation = useMutation({
    mutationFn: (decision: 'verified' | 'rejected') => {
      if (!verification) return Promise.reject(new Error('No verification selected'));
      return verificationsService.decide({ verification_id: verification.id, decision, notes });
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['verifications'] });
      toast.success('Verification recorded');
      onClose();
    },
    onError: (err) => toast.error('Could not record verification', errorMessage(err)),
  });

  return (
    <Modal
      open={Boolean(verification)}
      onClose={onClose}
      title="Decide verification"
      description={
        verification
          ? `Document ${verification.document_id.slice(0, 8)} · method ${verification.method || 'manual'} — currently ${verification.status}`
          : undefined
      }
      footer={
        <>
          <span className="mr-auto text-[13px] text-ink-500">Mark this document as verified or send it back.</span>
          <Button variant="outline" onClick={onClose} disabled={mutation.isPending}>Cancel</Button>
          <Button
            variant="danger"
            onClick={() => mutation.mutate('rejected')}
            loading={mutation.isPending && mutation.variables === 'rejected'}
          >
            Reject
          </Button>
          <Button
            onClick={() => mutation.mutate('verified')}
            loading={mutation.isPending && mutation.variables === 'verified'}
          >
            Verify
          </Button>
        </>
      }
    >
      <Textarea
        label="Notes (optional)"
        placeholder="Evidence, method, or reason…"
        value={notes}
        onChange={(e) => setNotes(e.target.value)}
        rows={4}
      />
    </Modal>
  );
}
