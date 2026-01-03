# Production Go Backend Service

This repository contains a Go-based backend service designed with
production-first principles.

The project includes:
- a Golang backend service (API layer)
- PostgreSQL database integration (AWS RDS)
- secure infrastructure setup for development and production parity
- database migrations and operational tooling

Infrastructure documentation is included to explain **how the backend is deployed and operated**, not as a standalone infra project.

---

## Architecture Overview

- **VPC** (private network)
- **Public Subnet**
  - Bastion EC2 (SSM access)
- **Private Subnet**
  - RDS PostgreSQL
- **Security Groups**
  - Bastion → RDS access on port 5432
- **Migration Tool**
  - Goose (SQL-based migrations)

---

## 1. AWS Infrastructure Setup

### 1.1 Create VPC

- CIDR: `10.0.0.0/16`
- Enable:
  - DNS Resolution
  - DNS Hostnames

---

### 1.2 Create Subnets

Minimum required:

- **Public Subnet** (for Bastion)
- **Private Subnet(s)** (for RDS)

Example:

- Public: `10.0.1.0/24`
- Private: `10.0.2.0/24`

---

### 1.3 Internet Gateway

- Create Internet Gateway
- Attach to VPC
- Route Table (Public Subnet):
  - `0.0.0.0/0 → Internet Gateway`

---

## 2. Security Groups

### 2.1 Bastion Security Group (`<your-project-bastion>`)

**Inbound**

- SSH (22) from your IP  
  _or_
- No inbound rules if using SSM only

**Outbound**

- **All traffic → 0.0.0.0/0**

> This is required so the bastion can reach RDS on port 5432.

---

### 2.2 RDS Security Group (`<your-project-db>`)

**Inbound**

- PostgreSQL (5432)
- Source: `<your-project-bastion>`

**Outbound**

- Default (allow all)

---

## 3. Create RDS PostgreSQL

- Engine: PostgreSQL
- Instance class: `db.t4g.micro`
- Public access: ❌ Disabled
- Subnet group: Private subnets only
- Security Group: `<your-project-db>`

> Note: **DB instance identifier ≠ database name**

---

## 4. Bastion Host Setup

### 4.1 Create EC2 Instance

- Instance type: `t3.micro` or `t4g.micro`
- Subnet: Public
- Security group: `<your-project-bastion>`
- IAM Role: `AmazonSSMManagedInstanceCore`

---

### 4.2 Access Bastion

Preferred:

- **AWS SSM Session Manager**

Alternative:

- SSH with key pair (dev only)

---

## 5. Cost Notes (Important)

Free Tier does **not** cover:

- NAT Gateways
- VPC Interface Endpoints

For development:

- Delete NAT Gateway
- Delete VPC Interface Endpoints when not needed
- Stop EC2 when idle

Expected dev cost: **$0–$5 / month**

---

## 6. Install PostgreSQL Client on Bastion

```bash
sudo dnf install -y postgresql15
```

---

## 7. Connect to RDS (Admin)

```bash
psql -h <RDS_ENDPOINT> -U postgres -d postgres -p 5432
```

---

## 8. Create Application Database User

```sql
CREATE USER <your_app> WITH PASSWORD 'STRONG_PASSWORD';

GRANT CONNECT ON DATABASE "<your-project-db>" TO <your_app>;

\c "<your-project-db>"

GRANT USAGE, CREATE ON SCHEMA public TO your_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO your_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO your_app;
```

---

## 9. Install Goose (Migration Tool)

```bash
curl -L https://github.com/pressly/goose/releases/download/v3.26.0/goose_linux_x86_64 -o goose
chmod +x goose
sudo mv goose /usr/local/bin/goose
goose --version
```

Check architecture if needed:

```bash
uname -m
```

---

## 10. Environment Variable

```bash
set +H
export DATABASE_URL='postgres://your_app:PASSWORD@<RDS_ENDPOINT>:5432/your-project-db?sslmode=require'
```

Verify:

```bash
psql "$DATABASE_URL" -c "select current_user, current_database();"
```

---

## 11. Migrations Directory

Always work inside HOME directory:

```bash
mkdir -p ~/migrations
```

## 12. PostgreSQL Extensions (RDS Rule)

Extensions must be installed using admin user:

```bash
psql -h <RDS_ENDPOINT> -U postgres -d your-project-db -p 5432 \
  -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"
```

## 13. Create Migration File (Safe Method)

```bash
tee ~/migrations/0001_init_schema.sql > /dev/null <<'SQL'
-- +goose Up
-- schema definitions
-- +goose Down
-- drop tables
SQL
```

Avoid sudo cat > file
Shell redirection happens before sudo.

## 14. Run Migrations

```bash
goose -dir ~/migrations postgres "$DATABASE_URL" up
```

Verify

```bash
psql "$DATABASE_URL" -c "\dt"
```

## 15. Best Practices

- Bastion is for bootstrap / emergency only
- Long-term migrations should run via:
      - CI/CD
      - ECS task / Kubernetes Job
- Keep migrations in the repository
- Do not rely on manual bastion access in production

## Final State

- Private RDS PostgreSQL
- Secure access via Bastion
- App-level DB user
- Goose migrations working
- Cost under control
- Ready for backend API development
