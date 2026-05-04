# Cursor pagination drops rows when timestamps tie

A change-data-capture listener pages through an `events` table using a
cursor that is the last seen `createdAt` timestamp. It works for the
steady-state stream of one event per millisecond.

It does **not** work whenever a writer commits multiple events in a
single transaction — they all get the same `createdAt`, and the listener
silently misses some of them. Customers report missing notifications
that always seem to arrive in clusters of 4–6 events at the same
millisecond.

Read `bug/notes.md` and `bug/page.ts`, then pick the explanation that
best matches the observed behaviour.
