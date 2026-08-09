# AeroXe DocuFlow

> Phase 1 | AeroXe Ecosystem

---

## Table of Contents

- [What Is This Project?](#what-is-this-project)
- [Why Was It Built?](#why-was-it-built)
- [When Should You Use It?](#when-should-you-use-it)
- [Where Does It Run?](#where-does-it-run)
- [How Does It Work?](#how-does-it-work)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Service Modules](#service-modules)
- [Saga Orchestrator](#saga-orchestrator)
- [WebSocket Events](#websocket-events)
- [Database Schema](#database-schema)
- [Setup & Installation](#setup--installation)
- [Environment Variables](#environment-variables)
- [Development](#development)
- [Testing](#testing)
- [Deployment](#deployment)

---

## What Is This Project?

AeroXe DocuFlow is a digital document management system that handles document upload, verification, approval workflows, version control, secure storage, and searchable archival.

---

## Why Was It Built?

Paper-based document management is inefficient, insecure, and makes retrieval difficult. DocuFlow digitizes documents with automated verification, approval workflows, and full-text search capabilities.

---

## When Should You Use It?

Use DocuFlow for legal firms, government agencies, enterprises, or any organization requiring systematic document management with verification and approval workflows.

---

## Where Does It Run?

Backend runs Go/Hertz with PostgreSQL for document metadata, Redis for search indexing, NATS for document processing events. S3-compatible storage for document files. WebSocket pushes document status updates and approval requests.

---

## How Does It Work?

DocuFlow implements DocumentService, VerificationService, ApprovalService, StorageService, TemplateService, SearchService, AnalyticsService, and ReportService. DocumentUpload saga handles: upload → virus scan → metadata extraction → storage → indexing. VerificationProcess saga validates document authenticity. ApprovalChain saga routes documents for multi-level approval.

---

## Architecture

### High-Level Architecture

```
+---------------------------------------------------------------------+
|                          CLIENT LAYER                               |
|  +----------+  +------------------+  +-----------------------+     |
|  | React    |  | Android          |  | iOS                   |     |
|  | (Web)    |  | (Kotlin+Compose) |  | (SwiftUI)             |     |
|  +----+-----+  +--------+---------+  +----------+------------+     |
|       |                 |                        |                  |
+-------+-----------------+------------------------+------------------+
        |                 |                        |
        v                 v                        v
+---------------------------------------------------------------------+
|                       API GATEWAY (Hertz)                           |
|  +-------------+  +--------------+  +------------------------+     |
|  | HTTP REST   |  | gRPC Proxy   |  | WebSocket Hub          |     |
|  | Routes      |  | (grpc-gw)    |  | (coder/websocket)      |     |
|  +------+------+  +------+-------+  +----------+------------+     |
|         |                |                       |                  |
|  +------v----------------v-----------------------v----------+      |
|  |  Auth | Rate Limit | Circuit Breaker | Logging           |      |
|  +----------------------------------------------------------+      |
+-------------+-------------------+------------------+---------------+
              |                   |                  |
     +--------v--------+  +------v------+  +--------v--------+
     |  gRPC (sync)    |  |  NATS (async)|  |  WebSocket      |
     |  point-to-point |  |  pub/sub     |  |  real-time      |
     +--------+--------+  +------+-------+  +--------+--------+
              |                   |                  |
+-------------v-------------------v------------------v---------------+
|                  MODULAR MONOLITH BACKEND                          |
|  +------------+ +------------+ +------------+ +------------+     |
|  | Module A   | | Module B   | | Module C   | | Module D   |     |
|  +-----+------+ +-----+------+ +-----+------+ +-----+------+     |
|        |              |              |              |              |
|  +-----v--------------v--------------v--------------v------+      |
|  |  +----------+ +-------+ +--------+ +--------------+    |      |
|  |  | Postgres | | Redis | |  NATS  | | Saga Engine  |    |      |
|  |  +----------+ +-------+ +--------+ +--------------+    |      |
|  +---------------------------------------------------------+      |
+--------------------------------------------------------------------+
```

### Tech Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **HTTP Framework** | [Hertz](https://github.com/cloudwego/hertz) | High-performance HTTP server |
| **RPC** | [gRPC](https://grpc.io/) | Synchronous service-to-service |
| **Messaging** | [NATS](https://nats.io/) JetStream | Async event-driven messaging |
| **Database** | [PostgreSQL](https://www.postgresql.org/) 15+ | Primary data store |
| **Cache** | [Redis](https://redis.io/) 7+ | Caching, sessions, saga state |
| **WebSocket** | [coder/websocket](https://github.com/coder/websocket) | Real-time communication |
| **Frontend** | React 18 + TypeScript + Tailwind | Web application |
| **Android** | Kotlin + Jetpack Compose + Hilt | Android application |
| **iOS** | Swift 5.9+ + SwiftUI | iOS application |

---

## Service Modules

| Module | Description | Protocol |
|--------|-------------|----------|
| DocumentService | Core document operations | gRPC + NATS |
| VerificationService | Core verification operations | gRPC + NATS |
| ApprovalService | Core approval operations | gRPC + NATS |
| StorageService | Core storage operations | gRPC + NATS |
| TemplateService | Core template operations | gRPC + NATS |
| SearchService | Core search operations | gRPC + NATS |
| AnalyticsService | Core analytics operations | gRPC + NATS |
| ReportService | Core report operations | gRPC + NATS |

---

## Saga Orchestrator

| Saga | Pattern |
|------|---------|
| DocumentUpload | Orchestrated via NATS + Redis state |
| VerificationProcess | Orchestrated via NATS + Redis state |
| ApprovalChain | Orchestrated via NATS + Redis state |
| Archival | Orchestrated via NATS + Redis state |

---

## WebSocket Events

| Event | Description |
|-------|-------------|
| `document_uploaded` | Real-time updates for document uploaded |
| `verification_needed` | Real-time updates for verification needed |
| `approval_required` | Real-time updates for approval required |
| `document_ready` | Real-time updates for document ready |

---

## Database Schema

| Table | Description | Key Fields |
|-------|-------------|------------|
| Document | `documents` table | UUID, timestamps, soft delete |
| Verification | `verifications` table | UUID, timestamps, soft delete |
| Approval | `approvals` table | UUID, timestamps, soft delete |
| Storage | `storages` table | UUID, timestamps, soft delete |
| Template | `templates` table | UUID, timestamps, soft delete |
| Version | `versions` table | UUID, timestamps, soft delete |
| Category | `categorys` table | UUID, timestamps, soft delete |
| Access | `accesss` table | UUID, timestamps, soft delete |
| Log | `logs` table | UUID, timestamps, soft delete |
| AuditLog | `auditlogs` table | UUID, timestamps, soft delete |

### Redis Usage

| Key Pattern | Purpose | TTL |
|------------|---------|-----|
| `session:<user_id>` | User session | 24h |
| `cache:<slug>:<id>` | Entity cache | 15m |
| `saga:<saga_id>` | Saga state | Until completion |
| `ratelimit:<ip>` | Rate limiting | 1m |

---

## Setup & Installation

### Prerequisites

- Go 1.22+
- PostgreSQL 15+
- Redis 7+
- NATS Server 2.10+ (with JetStream)
- Node.js 18+ (for React)
- Docker & Docker Compose (optional)

### Quick Start

```bash
# Clone the repository
git clone https://github.com/aeroxe/docu-flow.git
cd docu-flow

# Start infrastructure services
docker-compose up -d postgres redis nats

# Run database migrations
make migrate-up

# Seed initial data
make seed

# Start the Go server
make run

# In another terminal - start React frontend
cd clients/web
npm install
npm run dev
```

### Docker Compose

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: docu-flow
      POSTGRES_USER: aeroxe
      POSTGRES_PASSWORD: secret
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  nats:
    image: nats:2.10-alpine
    ports:
      - "4222:4222"
      - "8222:8222"
    command: ["-js"]

  app:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis
      - nats
```

---

## Environment Variables

```bash
# Server
PORT=8080
ENV=development
LOG_LEVEL=debug

# PostgreSQL
DATABASE_URL=postgres://aeroxe:secret@localhost:5432/docu-flow?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# NATS
NATS_URL=nats://localhost:4222

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRY=24h

# gRPC
GRPC_PORT=9090

# WebSocket
WS_MAX_CONNECTIONS=1000
WS_PING_INTERVAL=30s
```

---

## Development

### Makefile Targets

```makefile
run:
	go run cmd/server/main.go

build:
	go build -o bin/server cmd/server/main.go

test:
	go test ./... -v -cover

migrate-up:
	go run cmd/migrate/main.go up

proto-gen:
	protoc --go_out=. --go-grpc_out=. proto/*.proto

lint:
	golangci-lint run
```

### React Development

```bash
cd clients/web
npm install
npm run dev
npm run build
npm test
```

---

## Testing

```bash
# Unit tests
go test ./internal/modules/... -v

# Integration tests
go test ./internal/modules/... -tags=integration -v

# Coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Deployment

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: docu-flow
spec:
  replicas: 3
  selector:
    matchLabels:
      app: docu-flow
  template:
    metadata:
      labels:
        app: docu-flow
    spec:
      containers:
      - name: app
        image: aeroxe/docu-flow:latest
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

---

## License

Copyright (c) 2026 AeroXe Enterprises Private Limited. All rights reserved.

---

*Built with love by the AeroXe Team*
