import { useCallback } from 'react';
import { useAppDispatch } from '@/store';
import { dismissToast, pushToast } from '@/store/uiSlice';

export function useToast() {
  const dispatch = useAppDispatch();

  const success = useCallback(
    (title: string, message?: string) => {
      dispatch(pushToast({ kind: 'success', title, message }));
    },
    [dispatch],
  );

  const error = useCallback(
    (title: string, message?: string) => {
      dispatch(pushToast({ kind: 'error', title, message }));
    },
    [dispatch],
  );

  const info = useCallback(
    (title: string, message?: string) => {
      dispatch(pushToast({ kind: 'info', title, message }));
    },
    [dispatch],
  );

  const dismiss = useCallback(
    (id: number) => {
      dispatch(dismissToast(id));
    },
    [dispatch],
  );

  return { success, error, info, dismiss };
}

/** Extract a readable message from any thrown value. */
export function errorMessage(err: unknown, fallback = 'Something went wrong'): string {
  if (err instanceof Error && err.message) return err.message;
  return fallback;
}
