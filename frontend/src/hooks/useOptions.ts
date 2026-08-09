import { useQuery } from '@tanstack/react-query';
import { useDebounce } from './useDebounce';
import { optionsService, type OptionItem, type OptionKind } from '@/services/options.service';

const OPTIONS_STALE_MS = 60_000;

/**
 * Fetches id+name options for a dropdown. The search is debounced so typing
 * in a combobox queries the server at most once per 250ms. Results are cached
 * for a minute — option lists rarely change mid-session.
 */
export function useOptions(kind: OptionKind, search: string, enabled = true) {
  const debounced = useDebounce(search, 250);
  return useQuery<OptionItem[]>({
    queryKey: ['options', kind, debounced],
    queryFn: ({ signal }) => optionsService.list(kind, debounced, 20, signal),
    enabled: enabled && (debounced.length === 0 || debounced.length >= 2),
    staleTime: OPTIONS_STALE_MS,
    placeholderData: (prev) => prev,
  });
}
