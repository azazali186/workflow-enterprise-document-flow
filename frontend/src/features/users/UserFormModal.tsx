import { useEffect, useState, type FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Modal } from '@/components/ui/Modal';
import { Button } from '@/components/ui/Button';
import { Select, TextInput } from '@/components/ui/Field';
import { usersService } from '@/services/users.service';
import { rolesService } from '@/services/roles.service';
import { useToast, errorMessage } from '@/hooks/useToast';
import type { User } from '@/types/entities';

interface UserFormModalProps {
  open: boolean;
  onClose: () => void;
  user?: User | null;
}

const USER_STATUSES = ['active', 'locked', 'pending'];

export function UserFormModal({ open, onClose, user }: UserFormModalProps) {
  const qc = useQueryClient();
  const toast = useToast();
  const isEdit = Boolean(user);

  const [email, setEmail] = useState('');
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [phone, setPhone] = useState('');
  const [status, setStatus] = useState('active');
  const [roleIds, setRoleIds] = useState<string[]>([]);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const rolesQuery = useQuery({
    queryKey: ['roles', 'all'],
    queryFn: () => rolesService.list({ limit: 100, sort_by: 'name', sort_dir: 'asc' }),
  });
  const roleOptions = rolesQuery.data?.items ?? [];

  useEffect(() => {
    if (!open) return;
    setEmail(user?.email ?? '');
    setName(user?.name ?? '');
    setPassword('');
    setPhone(user?.phone ?? '');
    setStatus(user?.status ?? 'active');
    setRoleIds(user?.roles?.map((r) => r.id) ?? []);
    setErrors({});
  }, [open, user]);

  const mutation = useMutation({
    mutationFn: () =>
      isEdit
        ? usersService.update({ id: user!.id, name, phone, status, role_ids: roleIds })
        : usersService.create({ email, password, name, phone, role_ids: roleIds }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['users'] });
      toast.success(isEdit ? 'User updated' : 'User created');
      onClose();
    },
    onError: (err) => {
      toast.error(isEdit ? 'Could not update user' : 'Could not create user', errorMessage(err));
    },
  });

  const toggleRole = (id: string) => {
    setRoleIds((prev) => (prev.includes(id) ? prev.filter((r) => r !== id) : [...prev, id]));
  };

  const validate = (): boolean => {
    const e: Record<string, string> = {};
    if (!name.trim()) e.name = 'Name is required.';
    if (!isEdit && !email.trim()) e.email = 'Email is required.';
    if (!isEdit && email.trim() && !/^\S+@\S+\.\S+$/.test(email.trim())) e.email = 'Enter a valid email.';
    if (!isEdit && password.length < 8) e.password = 'Password must be at least 8 characters.';
    setErrors(e);
    return Object.keys(e).length === 0;
  };

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (!validate()) return;
    mutation.mutate();
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit user' : 'Create user'}
      description={isEdit ? 'Update profile, status, and roles.' : 'Add a new user to the workspace.'}
      footer={
        <>
          <Button variant="outline" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button type="submit" form="user-form" loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Create user'}
          </Button>
        </>
      }
    >
      <form id="user-form" onSubmit={onSubmit} className="space-y-4" noValidate>
        <TextInput
          label="Full name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          error={errors.name}
          placeholder="Ada Lovelace"
          required
        />
        <TextInput
          label="Email address"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          error={errors.email}
          placeholder="ada@company.com"
          disabled={isEdit}
          hint={isEdit ? 'Email cannot be changed.' : undefined}
          required
        />
        {!isEdit && (
          <TextInput
            label="Temporary password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            error={errors.password}
            placeholder="Min. 8 characters"
            required
          />
        )}
        <TextInput label="Phone (optional)" value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="+1 555 000 1234" />
        {isEdit && (
          <Select label="Status" value={status} onChange={(e) => setStatus(e.target.value)}>
            {USER_STATUSES.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </Select>
        )}

        <fieldset>
          <legend className="mb-2 text-[13px] font-medium text-ink-700">Roles</legend>
          {roleOptions.length === 0 ? (
            <p className="text-xs text-ink-400">No roles available.</p>
          ) : (
            <div className="space-y-1.5">
              {roleOptions.map((role) => (
                <label
                  key={role.id}
                  className="flex cursor-pointer items-center gap-2.5 rounded-lg border border-ink-200 px-3 py-2.5 transition-colors hover:border-primary-300 has-checked:border-primary-400 has-checked:bg-primary-50/50"
                >
                  <input
                    type="checkbox"
                    checked={roleIds.includes(role.id)}
                    onChange={() => toggleRole(role.id)}
                    className="size-4 rounded border-ink-300 accent-primary-600"
                  />
                  <span className="flex-1">
                    <span className="block text-sm font-medium text-ink-800">{role.name}</span>
                    {role.description && <span className="block text-xs text-ink-400">{role.description}</span>}
                  </span>
                  <span className="text-[11px] font-medium uppercase tracking-wide text-ink-300">{role.code}</span>
                </label>
              ))}
            </div>
          )}
        </fieldset>
      </form>
    </Modal>
  );
}
