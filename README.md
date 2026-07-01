<h1 align="center">AxCom</h1>

<p align="center">
  <a href="https://github.com/axiolon/axcom/actions/workflows/pipeline.yml"><img src="https://github.com/axiolon/axcom/actions/workflows/pipeline.yml/badge.svg" alt="CI/CD Pipeline"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/axiolon/axcom?filename=axcom-backend%2Fgo.mod" alt="Go Version"></a>
  <a href="https://goreportcard.com/report/github.com/axiolon/axcom"><img src="https://goreportcard.com/badge/github.com/axiolon/axcom" alt="Go Report Card"></a>
  <a href="https://opensource.org/licenses/Apache-2.0"><img src="https://img.shields.io/github/license/axiolon/axcom" alt="License"></a>
  <a href="https://github.com/axiolon/axcom/releases"><img src="https://img.shields.io/github/v/release/axiolon/axcom" alt="GitHub Release"></a>
  <a href="https://github.com/axiolon/axcom/issues"><img src="https://img.shields.io/github/issues/axiolon/axcom" alt="Issues"></a>
  <a href="https://github.com/axiolon/axcom/pulls"><img src="https://img.shields.io/github/issues-pr/axiolon/axcom" alt="Pull Requests"></a>
  <a href="https://hub.docker.com/"><img src="https://img.shields.io/docker/v/axiolon/axcom?sort=semver" alt="Docker Image Version"></a>
</p>

<p align="center">
  <strong>A modular, extensible ecommerce engine for building custom commerce platforms.</strong>
</p>

<p align="center">
  <a href="https://axcom.axiolon.com">Homepage</a> •
  <a href="https://axiolon.github.io/axcom/">Documentation</a>
</p>

---

AxCom is a high-performance, containerized e-commerce backend built with **Go**, utilizing **Gin**, **PostgreSQL** / **MongoDB**, **Redis**, and **OpenTelemetry**. It is designed with clean architecture principles to support massive scalability, secure checkout, real-time inventory management, and robust modular domain boundaries.

## 🚀 Quick Start

For detailed setup, configuration, and architecture guides, please refer to our official [Documentation](https://axiolon.github.io/axcom/).

### Prerequisites

- **Go** 1.25.11+
- **PostgreSQL** 15+ or **MongoDB** 6.0+
- **Redis** 7.0+
- **Docker** (Optional)

### Running Locally

```bash
# 1. Configure environment
cd axcom-backend
cp config.example.yaml config.yaml
cp .env.example .env.dev

# 2. Start the server
go run cmd/server/main.go
```

## 📖 Learn More

Our [documentation site](https://axiolon.github.io/axcom/) covers everything you need:

- 🏛️ **Architecture:** Learn about our modular clean architecture design.
- 🔌 **API Reference:** Detailed endpoint summaries, requests, and response models.
- 🧪 **Testing & Quality:** How to run unit tests, coverage, and linting checks.
- 🚀 **Deployment:** Guidelines for Kubernetes, Docker, and CI/CD pipelines.

---

## 📄 License

AxCom is released under the [Apache 2.0 License](LICENSE).
