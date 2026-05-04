export function reverseString(s: string): string {
  return [...s].reverse().join("");
}

export function isPalindrome(s: string): boolean {
  const cleaned = s.toLowerCase().replace(/[^a-z0-9]/g, "");
  return cleaned === [...cleaned].reverse().join("");
}

export function countVowels(s: string): number {
  let count = 0;
  for (const ch of s.toLowerCase()) {
    if (ch === "a" || ch === "e" || ch === "i" || ch === "o" || ch === "u") {
      count++;
    }
  }
  return count;
}
