# Metrics Model

Metrics are deterministic, versioned, evidence-backed, and computed over explicit half-open UTC
windows `[from,to)`. Every result carries unit, window, cutoff, definition version, coverage, and a
state distinguishing value from unknown/unavailable/insufficient/not-applicable.

Canonical cohorts exclude drafts, prereleases, bots, merges, and non-default-branch activity where
the accepted definition requires it. Issue first response, PR ready-to-merge duration, backlog
reconstruction, contributor concentration, health dimensions, comparisons, robust trends, and
forecasts follow the frozen algorithms in `01-product-requirements.md` and cases UT-228–UT-242.

Overall health uses equal fixed weights for available absolute dimensions and never redistributes a
missing dimension. Comparisons share one cutoff or mark results incomparable. Models may explain
these results but cannot calculate or replace them.
