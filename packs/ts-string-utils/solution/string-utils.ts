function graphemes(s: string): string[] {
  const seg = new Intl.Segmenter("en", { granularity: "grapheme" });
  const out: string[] = [];
  for (const piece of seg.segment(s)) {
    out.push(piece.segment);
  }
  return out;
}

export function reverseString(s: string): string {
  return graphemes(s).reverse().join("");
}

export function isPalindrome(s: string): boolean {
  const normalised = s
    .normalize("NFC")
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]/gu, "");
  const g = graphemes(normalised);
  for (let i = 0, j = g.length - 1; i < j; i++, j--) {
    if (g[i] !== g[j]) return false;
  }
  return true;
}

const VOWELS = new Set(["a", "e", "i", "o", "u"]);

export function countVowels(s: string): number {
  let count = 0;
  for (const g of graphemes(s)) {
    const decomposed = g.normalize("NFD");
    const base = decomposed.codePointAt(0);
    if (base === undefined) continue;
    const ch = String.fromCodePoint(base).toLowerCase();
    if (VOWELS.has(ch)) count++;
  }
  return count;
}
