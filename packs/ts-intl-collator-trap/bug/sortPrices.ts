// Sort string-encoded prices in natural numeric order.
//
// We use Intl.Collator with numeric: true so that "10" sorts after "2",
// not after "1" (the default lexicographic ordering).
const collator = new Intl.Collator("en", { numeric: true });

export function sortPrices(prices: string[]): string[] {
  return [...prices].sort(collator.compare);
}
