export function chunk<T>(items: T[], size: number): T[][] {
  if (size <= 0) return [];
  const out: T[][] = [];
  for (let i = 0; i < items.length; i += size) {
    out.push(items.slice(i, i + size));
  }
  return out;
}

export function unique<T>(items: T[]): T[] {
  const seen = new Set<T>();
  const out: T[] = [];
  for (const x of items) {
    if (!seen.has(x)) {
      seen.add(x);
      out.push(x);
    }
  }
  return out;
}

export function flatten<T>(items: T[][]): T[] {
  const out: T[] = [];
  for (const group of items) {
    for (const x of group) {
      out.push(x);
    }
  }
  return out;
}
