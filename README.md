# Golang Backend Platform (Production-Oriented)

This repository contains a **production-oriented backend platform** built with Golang + PostgreSQL
design to model a real-world business workflow, such as:
- Product & Price Management
- Customer categorization
- Transactions & fulfillment
- Stock Control
- Partial and Multi-stage payments

This project is intentionally structured to demonstrate **clean architecture, transactional correctness, and operational readiness**, not just CRUD Endpoints.

This is an upgrade and re-architecture of a previous tenant-based system, rebuilt to support:
- Strong domain boundaries
- Financial correctness
- Future frontend and integration work

---

## What This Backend Does

At a high level, the system supports the following real-world flow:
1. **Create Customers**
    - Customers and belong to pricing categories (e.g. Regular, Special or VIP)
2. **Create products**
    - Products have stock, prices, and active/inactive state
3. **Define prices per category**
    - One product can have different prices per customer category
4. **Create transactions**
    - Transactions contain multiple items
    - Prices are resolved based on customer category
5. **Fulfill transactions**
    - Stock is deducted at fulfillment time (not at order creation)
6. **Record payments**
    - Support cash or transfer (later maybe will connected to payment gateway too)
    - Supports partial payments
    - Transaction payment state is recalculated automatically

This mirrors how real shops and SMEs operate, especially in Indonesia.

---
## Core Domain Concepts

**Products**
  - Master data only
  - Stock is tracked directly on the product
  - Prices are stored separately for flexibility and history

**Customers**
  - Can belong to a category
  - Category determines effective price during transaction

**Transactions**
  - Created independently of payment
  - Immutable line items
  - Fulfillment controls stock deduction
  - Status-driven lifecycle

**Payments**
  - Multiple payments per transaction
  - Partial, full, or overpaid supported
  - Transaction payment status updates automatically

---
## Architecture Principles Used

This project follows some best practice and Clean Architecture-style separation, adapted pragmatically for Go:
```pgsql
cmd/
internal/
  delivery/        --> HTTP handlers
  usecase/         --> domain logic (Business rules)
  repository/      --> Database access (postgresql)
  config/          --> App Config
  db/              --> DB bootstrap
migrations/        --> Sql Migrations (Goose)
```

---
## API Coverage (Current)
Implemented endpoints include:
- Admin Auth (JWT)
- Product CRUD
- Product Price CRUD
- Customer CRUD
- Transaction creation & listing
- Transaction fulfillment
- Payment creation & listing
This is sufficient to support a real frontend.

---
## Project Status
- Core backend domain will be still updated.
- Ready for :
  - Frontend integration
  - Extended payment logic
  - CI/CD
  - Reporting
 
---
## Disclaimer
This repository is not a tutorial project.

It is a learning-driven for production system, built to reflect how real systems behave under real constraints.
