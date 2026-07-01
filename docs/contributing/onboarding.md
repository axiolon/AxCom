---
title: "Developer Onboarding"
description: "Step-by-step developer setup and onboarding guide for AxCom."
sidebar_position: 2
---

# Developer Onboarding Guide

This guide helps you set up a local development environment for AxCom, run tests, and check your work before committing code.

---

## 🛠️ Prerequisites

To build and run the AxCom backend, make sure you have the following installed:

- **Go 1.25.0+**
- **MongoDB 6.0+** (Local installation or running via Docker)
- **Redis 7.0+** (Local installation or running via Docker)
- **Node.js 18+** (Required only for editing/running the documentation site)
- **Docker** (Optional, but highly recommended for containerized testing)

---

## 🚀 Local Environment Setup

### 1. Clone the Repository

Clone the codebase and navigate to the project root:

```bash
git clone https://github.com/axiolon-labs/ecom-engine.git
cd ecom-engine
```

### 2. Configure Environment Variables

Copy the example environment configuration file to create your development configuration:

```bash
cp .env.example .env
```

Open `.env` in your text editor and verify the following settings are pointing to your local MongoDB and Redis instances:

```ini
MONGO_URI=mongodb://localhost:27017/ecom_engine
REDIS_URI=redis://localhost:6379/0
SERVER_PORT=8080
JWT_SECRET=your-local-development-secret-key-change-in-prod
```

### 3. Start Dependencies (Optional)

If you prefer running MongoDB and Redis via Docker Compose:

```bash
docker compose -f deployments/docker-compose.yml up -d
```

### 4. Run the Backend Server

Launch the server from the root directory:

```bash
go run cmd/server/main.go
```

The server will boot up and start listening on port `8080` (or whatever port is defined in `.env`).

---

## 🧪 Testing & Code Quality

Before opening a pull request, you must ensure that all code meets quality standards and tests pass.

### 1. Code Style and Linting

We use `golangci-lint` to maintain strict code guidelines. Check the rules defined in [.golangci.yml](../../.golangci.yml) and run the linter:

```bash
golangci-lint run
```

### 2. Running Unit & Integration Tests

Ensure all core tests run and pass without race conditions:

```bash
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
```

### 3. Running E2E Tests

To run full end-to-end tests (requires active test instances of Redis and MongoDB):

```bash
cd axcom-backend
go test -tags e2e -v -timeout 300s ./tests/e2e/...
```

---

## 📖 Running the Documentation Site Locally

The documentation website is built with Docusaurus. If you are updating documentation under `docs/`, you should preview the site locally to check for layout or link issues.

### 1. Install Documentation Dependencies

Navigate to the docs directory and install package requirements:

```bash
cd axcom-engine-docs-site
npm install --legacy-peer-deps
```

### 2. Generate API Reference Pages

If the OpenAPI schema has updated, generate the new API documentation:

```bash
npm run clean-api-docs
npm run gen-api-docs
```

### 3. Run the Documentation Server

Start the local development server:

```bash
npm run start
```

Open [http://localhost:3000](http://localhost:3000) in your browser to view your local copy of the documentation.

### 4. Build Validation

Ensure the documentation builds successfully without any broken links or MDX errors before pushing your branch:

```bash
npm run build
```
