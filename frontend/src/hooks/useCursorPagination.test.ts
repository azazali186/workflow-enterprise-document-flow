import { describe, expect, it } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useCursorPagination } from './useCursorPagination';
import type { PaginationMeta } from '@/types/api';

function meta(totalCount: number, nextCursor?: string): PaginationMeta {
  return { total_count: totalCount, next_cursor: nextCursor ?? '', has_more: Boolean(nextCursor), limit: 20, returned_count: 0 };
}

describe('useCursorPagination', () => {
  it('summarizes the first window as 1..limit', () => {
    const { result } = renderHook(() => useCursorPagination(20));
    act(() => result.current.applyResult({ pagination: meta(245, 'c1') }));
    expect(result.current.summary).toBe('Showing 1–20 of 245');
  });

  it('advances the window after next()', () => {
    const { result } = renderHook(() => useCursorPagination(20));
    act(() => result.current.applyResult({ pagination: meta(245, 'c1') }));
    act(() => result.current.next('c1'));
    act(() => result.current.applyResult({ pagination: meta(245, 'c2') }));
    expect(result.current.summary).toBe('Showing 21–40 of 245');
  });

  it('caps the last window at the total', () => {
    const { result } = renderHook(() => useCursorPagination(20));
    act(() => result.current.applyResult({ pagination: meta(245, 'c1') }));
    // Page 1 of 13. 13 pages * 20 = 260 > 245 → the final window is 241..245.
    for (let i = 1; i < 13; i++) {
      act(() => result.current.next(`c${i}`));
    }
    expect(result.current.summary).toBe('Showing 241–245 of 245');
  });

  it('tracks canPrev only after navigating forward', () => {
    const { result } = renderHook(() => useCursorPagination(20));
    expect(result.current.canPrev).toBe(false);
    act(() => result.current.applyResult({ pagination: meta(50, 'c1') }));
    act(() => result.current.next('c1'));
    expect(result.current.canPrev).toBe(true);
    act(() => result.current.prev());
    expect(result.current.canPrev).toBe(false);
  });

  it('reset() returns to the first page and zeroes the total', () => {
    const { result } = renderHook(() => useCursorPagination(20));
    act(() => result.current.applyResult({ pagination: meta(50, 'c1') }));
    act(() => result.current.next('c1'));
    act(() => result.current.reset());
    expect(result.current.canPrev).toBe(false);
    expect(result.current.summary).toBe('');
  });

  it('ignores next() without a cursor', () => {
    const { result } = renderHook(() => useCursorPagination(20));
    act(() => result.current.applyResult({ pagination: meta(10) }));
    act(() => result.current.next(undefined));
    expect(result.current.summary).toBe('Showing 1–10 of 10');
  });

  it('empty dataset yields no summary', () => {
    const { result } = renderHook(() => useCursorPagination(20));
    act(() => result.current.applyResult({ pagination: meta(0) }));
    expect(result.current.summary).toBe('');
  });
});
