# golang-backend-platform

A production-oriented backend built in Go — not to learn from a tutorial, but to solve a real problem.

---

## Why I built this

A friend runs a small shop. They needed something to track products, manage stock, record transactions, and handle payments from customers who often pay in installments. They were doing it manually — spreadsheets, paper, memory.

I built this for them. It runs on a local network and they still use it today.

But I also built it to prove something to myself: that I could design a backend that reflects how a real business actually operates, not just how a tutorial says it should.

---

## The core insight

Most beginner projects deduct stock when an order is created. That's wrong for how real shops work.

In a real shop, you don't lose the item when someone says "I want to buy this." You lose it when you physically hand it over. A customer might order, then cancel. Stock that was "deducted" on order creation becomes a ghost — your numbers are wrong and you don't know why.

So in this system:

- **Draft** → order exists, no stock touched
- **Pending** → stock is _reserved_ (set aside, not deducted)
- **Fulfilled** → stock is _committed_ (actually deducted from inventory)
- **Cancelled** → reserved stock is _released_ back

This mirrors how the shop owner thinks. It also means your stock numbers are always trustworthy.

---

## The other insight: partial payments

Indonesian SMEs don't always pay in full upfront. Customers pay what they can, when they can. A system that only knows "paid" or "unpaid" doesn't model reality.

So payments are tracked separately from transactions. Each transaction can have multiple payments. The system automatically recalculates payment status after every payment:

- `unpaid` → no payments yet
- `partial` → some payments made, but not enough
- `paid` → exactly covered
- `overpaid` → customer paid more than the total (happens more than you'd think)

---

## Architecture

I used Clean Architecture — not because the internet says to, but because it solved a real problem for me: I wanted to test my business logic without a running database.

```
cmd/
internal/
  delivery/      → HTTP handlers (Fiber) — how the outside world talks to the app
  usecase/       → business logic — the rules of the domain
  repository/    → database access (PostgreSQL via pgx)
  config/        → environment and app config
migrations/      → SQL migrations (Goose)
```

The layers only talk in one direction: delivery calls usecase, usecase calls repository. The usecase layer defines interfaces — it doesn't know or care whether the repository is PostgreSQL, an in-memory mock, or something else entirely.

This is why I can run 49 unit tests in under a second with zero database involvement.

---

## Key design decisions

**Stock lives on the product, not the transaction**
Stock is tracked directly on `products` as `stock_on_hand` and `stock_reserved`. When you reserve stock for a pending transaction, `stock_reserved` goes up. When you fulfill, `stock_on_hand` goes down and `stock_reserved` goes down. Available stock at any point = `stock_on_hand - stock_reserved`.

**Category-based pricing**
One product can have different prices for different customer categories (Regular, VIP, Wholesale). Price is resolved at transaction creation time based on the customer's category, with a fallback to the default price if no category-specific price exists.

**Pack size support**
Some products are sold in packs of a base unit — for example, you sell "Box of 12" but your stock is tracked in individual units. `pack_size` handles this conversion so stock deduction is always in base units, regardless of what the customer is buying.

**Payments are independent of transactions**
A transaction is created without any payment. Payments are recorded separately and can be partial. The transaction's `payment_status` is recomputed automatically using a single SQL CTE every time a new payment is added — no application-level logic needed for the recalculation.

---

## Testing strategy

Two layers of tests, each with a different job:

**Unit tests** (`internal/usecase/...`)
Test business logic in complete isolation using a mock store. No database, no network, no setup. All 49 tests run in under one second. These catch logic bugs — wrong status transitions, missing validation, incorrect stock operation sequencing.

**Integration tests** (`internal/repository/...`)
Test the real SQL against a real PostgreSQL database. These catch database bugs — wrong queries, constraint violations, transaction isolation issues. Run via Docker Compose locally or GitHub Actions CI in the pipeline.

```bash
# Unit tests (no DB needed)
go test ./internal/usecase/... -v

# Integration tests (requires DATABASE_URL)
go test ./internal/repository/... -v -count=1 -tags=integration
```

---

## Running locally

**Prerequisites:** Go 1.24+, Docker

```bash
# Clone the repo
git clone https://github.com/riolentius/golang-backend-platform.git
cd golang-backend-platform

# Copy env file and fill in your values
cp .env.sample .env

# Start the database
docker compose up db -d

# Run migrations
goose -dir ./migrations postgres "$DATABASE_URL" up

# Run the server
go run ./cmd/api
```

Or run the whole stack at once:

```bash
docker compose up
```

---

## Tech stack

| Layer            | Choice         |
| ---------------- | -------------- |
| Language         | Go 1.24        |
| HTTP Framework   | Fiber v2       |
| Database         | PostgreSQL     |
| DB Driver        | pgx/v5         |
| Migrations       | Goose          |
| Auth             | JWT            |
| Containerization | Docker         |
| CI               | GitHub Actions |

---

## Current status

The backend is complete and covers the full business workflow. It is actively used by a real user on a local network.

Planned next:

- Admin UI (frontend)
- OpenAPI / Swagger documentation
- Extended reporting
- Payment gateway integration

---

## What I learned

Before this project, I mostly learned Go from documentation and small scripts. This project forced me to make real decisions under real constraints — not "what does the tutorial say" but "what actually makes sense for this business."

The biggest shift was thinking about the domain first. I've spent years as a Business Analyst understanding how businesses work before writing a single line of code. That background made a real difference here. The stock reservation logic, the partial payment model, the category-based pricing — none of that came from a Go tutorial. It came from understanding the problem.

That's the kind of engineer I'm trying to be.
