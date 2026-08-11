# SYMBOLS

Running ledger of exported Go signatures. **Append after every merged file.**
This is the paste-source for the `AVAILABLE SYMBOLS` block in build orders
(`docs/08-BUILD-ORDERS.md §5`).

Signatures only — no bodies, no comments. Grouped by package, ordered by the
milestone that introduced them.

Regenerate a package's block with:

```bash
go doc -all ./internal/<pkg> | grep -E '^(func|type|const|var) '
```

---

## package domain

_(M2.2 — pending)_

## package resolve

_(M2 — pending)_

## package store

_(M3 — pending)_

## package content

_(M6 — pending)_

## package ingest

_(M7 — pending)_

## package api

_(M8 — pending)_
