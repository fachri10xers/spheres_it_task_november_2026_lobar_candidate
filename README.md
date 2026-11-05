# SPHERES — Fullstack Engineer Take-Home Assessment

## About the Project

SPHERES builds digital tools for primary health care in Lombok Barat and Purbalingga, Indonesia. The system supports clinic staff with patient registration, outpatient visits, pharmacy dispensing, billing, and reporting to the district health office (Dinkes). You do not need any prior knowledge of healthcare standards to complete this assessment.

> **All data in this assessment is synthetic.** Names, NIKs, addresses, and clinical records were generated with Faker. Any resemblance to real people is coincidental.

---

## What to Do

**Task 1 is required.** Then choose **one** of Task 2 or Task 3.

| Task | Focus | Time Budget |
|------|-------|-------------|
| **Task 1 — Clinic Sync Service** (required) | Review and fix a FastAPI service that syncs clinic data to an insurance claim API | ~5 hours |
| **Task 2 — Offline-First Sync** (choose this or Task 3) | System design + implement a server-side sync endpoint for offline mobile devices | ~6 hours |
| **Task 3 — Dinkes Report** (choose this or Task 2) | Produce monthly patient/visit numbers from a messy SQLite dataset | ~5 hours |

Pick by strength: **Task 2** suits people who like system design and backend work. **Task 3** suits people who like data and SQL.

Doing all three earns no extra credit. We respect your time — pick two and do them well.

---

## Time Window

You have **3 days** from the time you receive this assessment. Expected effort is **~10–11 hours** total.

---

## AI Policy

AI tools (ChatGPT, Copilot, Claude, etc.) are **allowed**. Each submission must include an `AI_USAGE.md` covering:

- Which tools you used
- What you used them for
- What you changed or rejected from AI output, and why

---

## What We Value

- **Clear reasoning** over volume. A short, well-argued answer beats a long one.
- **Prioritization.** Tell us what matters most and why.
- **Stated assumptions.** When the requirements are ambiguous, say what you assumed and why.

---

## Submission

Submit a GitHub repository (private) or a zip file containing:

```
your-submission/
  task1/                # Required
  task2/ or task3/      # Your chosen optional task
  CHOICE.md             # One line: which optional task you picked and why
  AI_USAGE.md           # AI tool usage disclosure
```

Reply to the assessment invitation email with a link to your repository or attach your zip file. Use subject line `SPHERES IT Assessment — [Your Name]`.

For clarifying questions (max one per task), reply to the invitation email.

---

## After Submission

We will schedule a **45-minute follow-up call** to walk through your work. This call counts for **30% of the overall score**. We will ask you to explain your decisions, modify code live, and discuss the task you did not choose.

---

## Getting Started

Each task folder has its own `README.md` with detailed instructions and a `SETUP.md` with setup steps.

### Prerequisites

- **Docker Desktop** (or Docker Engine with Compose v2) **or** Python 3.11+
- An SQLite client for Task 3: `sqlite3` CLI, DBeaver, or DB Browser for SQLite

### Quick Start

- **Task 1** runs with `docker compose up` (Docker) or `make run` (local Python). See `task1-clinic-sync/SETUP.md`.
- **Task 3** needs only the SQLite file and any query tool. See `task3-dinkes-report/SETUP.md`.
- **Task 2** is a design + implementation task — you build the endpoint. See `task2-offline-sync/README.md`.

### If Something Is Broken

If setup takes you more than 15 minutes, email us. That's our problem, not yours, and it won't count against you.
