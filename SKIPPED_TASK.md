# SKIPPED_TASK.md - Approach for Task 3 (Dinkes Monthly Report)

I chose Task 2, so Task 3 was skipped. I wrote this without the full analysis, grounded in one quick read-only profile of `clinic_data.sqlite` (`mode=ro`, SELECTs only); each inference would be validated further before any figure is published.

## What I already found while profiling (and the decision each finding forces)

- **`PKM-002-OLD` is a fourth clinic holding 2,420 visit rows but zero patient rows of its own** - it looks like a legacy identifier of `PKM-002` (which holds 2,298 visits). Merging vs separating nearly doubles PKM-002's column, so I would state the merge as an explicit assumption - and it is my one clarifying question to Dinkes, below.
- **Duplicate patients are real, not theoretical:** 152 nonzero NIKs are shared by 2+ patients (187 such pairs differ by name), and 303 patients carry an all-zero NIK. Same-NIK pairs never span clinics in this dataset, so I would de-duplicate within a clinic by shared NIK, flag differing names for review, and make no merge decision on all-zero NIKs (no identity evidence).
- **Every registration has a visit row - including 680 on registrations marked `cancelled` (372) or `no_show` (308)** - the status column contradicts the visit trail. For "total visits" I would count the care actually recorded (18,724) and disclose the 3.6% caveat in `ASSUMPTIONS.md`.
- **Timestamps are mixed-format:** 17,677 naive ISO strings (19 chars) vs **1,047 UTC rows with a `Z` suffix - all at `PKM-003`**, the one WIB clinic. I would parse both, normalize to each clinic's local date (WITA UTC+8, WIB UTC+7) before bucketing into Jan-Jun 2026, first verifying what clock the naive rows use via a plausibility check against the Z rows.
- **Structural integrity is clean** - no orphan rows; visits match registrations 1:1 - so the difficulty would be definitional (identity, consolidation, time), not relational; that is why I profiled before drafting.

## How I would produce the deliverables

Pin the report definitions ("unique" per what scope, cancelled-status visits counted or not); then one top-to-bottom runnable SQL script - CTEs for time normalization, clinic consolidation (`PKM-002-OLD`), and NIK-based dedup - producing the result table (months x Puskesmas, unique patients + total visits) computed both with and without merges to show sensitivity, plus `ASSUMPTIONS.md` mapping every decision to its evidence.

## The one clarifying question I would send

"Is `PKM-002-OLD` the legacy identifier of `PKM-002` to merge for the dashboard, or a separate service point during part of the period?" - merging or not changes one clinic's monthly numbers by roughly 2x and the table's row count, so it changes the answer rather than confirming a preference. (Follow-up only if room: the 680 cancelled/no_show visits.)
