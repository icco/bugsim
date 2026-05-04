# List utilities (TypeScript)

Implement the three exported functions in `src/list-utils.ts`:

- `chunk<T>(items: T[], size: number): T[][]` — split `items` into consecutive groups of `size`. The last group may be smaller. If `size <= 0`, return `[]`.
- `unique<T>(items: T[]): T[]` — return `items` with duplicates removed, preserving the order of first occurrence.
- `flatten<T>(items: T[][]): T[]` — concatenate all sub-arrays into a single array (one level deep).

Run the hidden tests with `r`.
