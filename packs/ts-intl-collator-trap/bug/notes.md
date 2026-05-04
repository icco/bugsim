# Reported behaviour

- Input: `["-3", "1", "-2", "2"]`
- Expected (natural numeric order): `["-3", "-2", "1", "2"]`
- Actual: `["-2", "-3", "1", "2"]`

The "Top losers" panel sorts ascending, so it ends up showing the
**second-largest** loss as the worst loser (-2 displayed before -3).

Re-running the sort with the same input always returns the same
(incorrect) order — this is not a stability bug, it's deterministic.
