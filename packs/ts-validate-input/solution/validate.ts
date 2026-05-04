// Linear-time email validator: inspect characters once, no nested
// quantifiers, no `*` after grouping that could backtrack.
export function isValidEmail(s: string): boolean {
  if (s.length === 0 || s.length > 254) return false;

  const at = s.indexOf("@");
  if (at < 0 || at !== s.lastIndexOf("@")) return false;

  const local = s.slice(0, at);
  const domain = s.slice(at + 1);

  if (!isValidLocal(local)) return false;
  if (!isValidDomain(domain)) return false;
  return true;
}

function isValidLocal(local: string): boolean {
  if (local.length === 0 || local.length > 64) return false;
  if (local.startsWith(".") || local.endsWith(".")) return false;
  if (local.includes("..")) return false;
  for (let i = 0; i < local.length; i++) {
    const c = local.charCodeAt(i);
    if (
      !(
        (c >= 0x30 && c <= 0x39) || // 0-9
        (c >= 0x41 && c <= 0x5a) || // A-Z
        (c >= 0x61 && c <= 0x7a) || // a-z
        c === 0x2e || // .
        c === 0x2b || // +
        c === 0x2d || // -
        c === 0x5f // _
      )
    ) {
      return false;
    }
  }
  return true;
}

function isValidDomain(domain: string): boolean {
  if (domain.length === 0 || domain.length > 253) return false;
  if (domain.startsWith(".") || domain.endsWith(".")) return false;
  const labels = domain.split(".");
  if (labels.length < 2) return false;
  for (let i = 0; i < labels.length; i++) {
    const label = labels[i];
    if (!isValidLabel(label)) return false;
    const isTld = i === labels.length - 1;
    if (isTld) {
      if (label.length < 2) return false;
      for (const ch of label) {
        if (!/[A-Za-z]/.test(ch)) return false;
      }
    }
  }
  return true;
}

function isValidLabel(label: string): boolean {
  if (label.length === 0 || label.length > 63) return false;
  if (label.startsWith("-") || label.endsWith("-")) return false;
  for (let i = 0; i < label.length; i++) {
    const c = label.charCodeAt(i);
    if (
      !(
        (c >= 0x30 && c <= 0x39) ||
        (c >= 0x41 && c <= 0x5a) ||
        (c >= 0x61 && c <= 0x7a) ||
        c === 0x2d
      )
    ) {
      return false;
    }
  }
  return true;
}

export function normalizePhone(s: string): string | null {
  // Strip allowed formatting; bail on anything else.
  let cleaned = "";
  for (const ch of s) {
    if (ch === " " || ch === "." || ch === "-" || ch === "(" || ch === ")") {
      continue;
    }
    cleaned += ch;
  }
  if (cleaned.startsWith("+1")) cleaned = cleaned.slice(2);
  else if (cleaned.startsWith("1") && cleaned.length === 11) cleaned = cleaned.slice(1);
  if (!/^\d{10}$/.test(cleaned)) return null;
  return `+1${cleaned}`;
}
