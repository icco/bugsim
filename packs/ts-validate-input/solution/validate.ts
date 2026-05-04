export function isValidEmail(s: string): boolean {
  if (/\s/.test(s)) return false;
  const parts = s.split("@");
  if (parts.length !== 2) return false;
  const [local, domain] = parts;
  if (local.length === 0) return false;
  if (!domain.includes(".")) return false;
  const [, ...rest] = domain.split(".");
  return rest.every((p) => p.length > 0);
}

export function normalizePhone(s: string): string | null {
  const digits = s.replace(/[\s().-]/g, "");
  if (!/^\d{10}$/.test(digits)) return null;
  return digits;
}
