# List utilities (TypeScript) — the lodash bug tour

Implement the three exported functions in `src/list-utils.ts`. The hidden
tests are taken from real bugs filed against `lodash`, so the gotchas are
specific:

- `chunk<T>(items: T[], size: number): T[][]` — split into consecutive
  groups of `size`. The last group may be smaller. **Edges:**
  - `size <= 0` returns `[]`.
  - `size > items.length` returns `[items]` (one chunk containing
    everything), **not** `[]`. (See `lodash#896` for the original bug.)

- `unique<T>(items: T[]): T[]` — return `items` with duplicates removed,
  preserving the order of first occurrence. **Edges:**
  - `NaN` should dedupe to a single entry, even though `NaN !== NaN`
    (use SameValueZero, like `Set`).
  - Equality is reference identity for objects: two structurally-equal
    objects are kept as distinct entries.
  - The input array must not be mutated.

- `flatten<T>(items: T[][]): T[]` — concatenate one level deep into a
  single array. **Edges:**
  - Must not throw `RangeError: Maximum call stack size exceeded` on
    inputs with hundreds of thousands of inner arrays. (See `lodash#349`:
    `[].concat(...big)` blows up because V8's argument limit is ~125k.)

Run the hidden tests with `r`.
