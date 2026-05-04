// In-memory implementation of the production cursor-pagination logic.
// The real version uses SQL: `WHERE created_at > $cursor ORDER BY
// created_at ASC LIMIT $pageSize`. The shape of the bug is the same.
type Event = { id: string; createdAt: number; payload: unknown };

export function pageAfter(
  rows: Event[],
  cursor: number | null,
  pageSize: number,
): Event[] {
  const after = cursor ?? Number.NEGATIVE_INFINITY;
  return rows
    .filter((r) => r.createdAt > after)
    .sort((a, b) => a.createdAt - b.createdAt)
    .slice(0, pageSize);
}

export function nextCursor(page: Event[]): number | null {
  const last = page[page.length - 1];
  return last ? last.createdAt : null;
}
