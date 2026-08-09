import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Modal } from '@/components/ui/Modal';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { rolesService, permissionsService } from '@/services/roles.service';
import { useToast, errorMessage } from '@/hooks/useToast';
import { groupByEntity } from '@/lib/permissionGroups';
import type { Permission, Role } from '@/types/entities';

interface PermissionsModalProps {
  open: boolean;
  onClose: () => void;
  role: Role | null;
}

export function PermissionsModal({ open, onClose, role }: PermissionsModalProps) {
  const qc = useQueryClient();
  const toast = useToast();
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const permsQuery = useQuery({
    queryKey: ['permissions', 'all'],
    queryFn: () => permissionsService.list({ limit: 500, sort_by: 'path', sort_dir: 'asc' }),
    enabled: open,
  });
  const allPerms = permsQuery.data?.items ?? [];
  const groups = useMemo(() => groupByEntity(allPerms), [allPerms]);

  useEffect(() => {
    if (open && role) {
      setSelected(new Set(role.permissions?.map((p) => p.id) ?? []));
    }
  }, [open, role]);

  const mutation = useMutation({
    mutationFn: () => {
      if (!role) return Promise.reject(new Error('No role selected'));
      return rolesService.assignPermissions(role.id, [...selected]);
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['roles'] });
      toast.success('Permissions updated');
      onClose();
    },
    onError: (err) => toast.error('Could not update permissions', errorMessage(err)),
  });

  const toggle = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleGroup = (perms: Permission[], on: boolean) => {
    setSelected((prev) => {
      const next = new Set(prev);
      for (const p of perms) {
        if (on) next.add(p.id);
        else next.delete(p.id);
      }
      return next;
    });
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={`Permissions · ${role?.name ?? ''}`}
      description="Select the API routes this role may call. Changes replace the current set."
      size="lg"
      footer={
        <>
          <span className="mr-auto text-[13px] text-ink-500">
            {selected.size} of {allPerms.length} selected
          </span>
          <Button variant="outline" onClick={onClose} disabled={mutation.isPending}>Cancel</Button>
          <Button onClick={() => mutation.mutate()} loading={mutation.isPending}>
            Save permissions
          </Button>
        </>
      }
    >
      {permsQuery.isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="h-10 animate-pulse rounded-lg bg-ink-100" />
          ))}
        </div>
      ) : (
        <div className="space-y-5">
          {groups.map((group) => (
            <div key={group.entity}>
              <div className="mb-2 flex items-center justify-between">
                <p className="text-[13px] font-semibold capitalize text-ink-800">{group.entity}</p>
                <div className="flex gap-2 text-xs">
                  <button className="text-primary-600 hover:text-primary-700 cursor-pointer" onClick={() => toggleGroup(group.perms, true)}>
                    Select all
                  </button>
                  <span className="text-ink-200">·</span>
                  <button className="text-ink-400 hover:text-ink-600 cursor-pointer" onClick={() => toggleGroup(group.perms, false)}>
                    None
                  </button>
                </div>
              </div>
              <div className="flex flex-wrap gap-2">
                {group.perms.map((p) => (
                  <button
                    key={p.id}
                    onClick={() => toggle(p.id)}
                    className={`rounded-lg border px-2.5 py-1.5 text-xs font-medium transition-all duration-100 cursor-pointer ${
                      selected.has(p.id)
                        ? 'border-primary-400 bg-primary-50 text-primary-700'
                        : 'border-ink-200 bg-white text-ink-500 hover:border-ink-300 hover:text-ink-700'
                    }`}
                  >
                    {p.method} {p.path.replace('/api/v1', '')}
                  </button>
                ))}
              </div>
            </div>
          ))}
          {allPerms.length === 0 && <Badge tone="neutral">No permissions synced yet.</Badge>}
        </div>
      )}
    </Modal>
  );
}
