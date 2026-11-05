# Task 3: Dinkes Monthly Report

> **Setup:** See [SETUP.md](SETUP.md) for how to open the data.

## Scenario

Dinkes (the district health office) wants **monthly unique patients and total visits per Puskesmas** for their dashboard. The data covers 6 months (January-June 2026) across the Puskesmas in the system.

## Your Task

1. Write the **SQL** (or script) that produces the numbers.
2. Include the **result table** in your submission.
3. Write `ASSUMPTIONS.md`: every data problem you found, how you handled it, and what you would confirm with Dinkes before this goes on a dashboard.
4. You may email **one clarifying question** during the 3-day window to `[EMAIL_ADDRESS]`.

## What We're Looking For

- Recognizing that the data is not as clean as it looks.
- Making decisions about how to handle ambiguity, and stating those decisions clearly.
- Correct numbers under your stated assumptions.
- A clarifying question that would change the answer, not just confirm it.

## Data

The file `clinic_data.sqlite` contains:

- `clinics` - facility information
- `patients` - patient records
- `registrations` - visit registrations with status
- `visits` - outpatient visit records
- `dispensing` - pharmacy dispensing records

See `SCHEMA.md` for column descriptions.

> **All data is synthetic.** Any resemblance to real people is coincidental.
