# Input validation (TypeScript)

Implement two helpers in `src/validate.ts`:

- `isValidEmail(s: string): boolean` — true when `s` looks like a basic email of the form `local@domain.tld`. The local part must be non-empty, the domain must contain a dot, and `s` must contain exactly one `@`. Whitespace anywhere is invalid.
- `normalizePhone(s: string): string | null` — accept a string that may include spaces, dashes, dots, or parentheses, and return the digits-only form if it is exactly 10 digits long (US phone). Return `null` for any other input.

The hidden tests focus on edge cases. Press `r` to run them.
