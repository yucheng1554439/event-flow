# Contributing to EventFlow

Thank you for your interest in EventFlow. This document explains how to set up a development environment, run tests, and submit changes.

## Code of Conduct

Be respectful and constructive. Focus on technical merit and clarity.

## Getting Started

### Prerequisites

| Tool | Version |
|------|---------|
| Go | 1.22+ |
| Docker & Docker Compose | Latest stable |
| Make | GNU Make or compatible |
| (Optional) Helm | 3.x |
| (Optional) Terraform | 1.5+ |

### Clone and Build

```bash
git clone https://github.com/yucheng1554439/event-flow.git
cd event-flow
make build
```

### Local Stack

```bash
make docker-up          # Kafka, PostgreSQL, Redis, services, Prometheus, Grafana
make migrate            # Apply migrations (requires DATABASE_URL)
```

API gateway: `http://localhost:8080` · gRPC: `localhost:9090` · Grafana: `http://localhost:3000`

## Development Workflow

1. **Fork** the repository and create a feature branch from `main`.
2. **Make changes** in small, focused commits with clear messages.
3. **Run checks** locally before opening a pull request (see below).
4. **Open a PR** with a summary of what changed and why.

### Branch Naming

- `feat/short-description` — new features
- `fix/short-description` — bug fixes
- `docs/short-description` — documentation only
- `ci/short-description` — pipeline changes

### Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(topic): add retention policy to create request
fix(retry): route to DLQ when max attempts exceeded
docs(readme): update quick start for Windows
ci(integration): share Testcontainers in TestMain
```

## Running Tests

```bash
make test                 # Unit tests (all packages)
make lint                 # go vet
make test-integration     # Integration tests (Docker required)
```

Integration tests use [Testcontainers](https://golang.testcontainers.org/) and need a running Docker daemon.

### Proto Regeneration

After editing files under `api/proto/`:

```bash
make proto
# or on Windows:
powershell -File scripts/generate-proto.ps1
```

Commit both `.proto` sources and generated `api/gen/go/` files.

## Project Layout

| Path | Purpose |
|------|---------|
| `cmd/` | Service entrypoints (api-gateway, consumer-worker, workflow-engine) |
| `internal/` | Application logic (not importable outside the module) |
| `pkg/` | Shared libraries (kafka, config, metrics, models) |
| `api/` | OpenAPI spec and protobuf definitions |
| `migrations/` | PostgreSQL schema |
| `docker/` | Dockerfiles and Compose stacks |
| `helm/` | Kubernetes Helm chart |
| `terraform/` | AWS infrastructure modules |
| `tests/` | Integration and load tests |
| `scripts/` | Demo, seeding, and tooling scripts |
| `docs/` | Architecture, deployment, and portfolio documentation |

## Pull Request Checklist

- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] Integration tests pass if you touched `internal/`, `pkg/`, or `tests/`
- [ ] OpenAPI / proto updated if API surface changed
- [ ] `CHANGELOG.md` updated under `[Unreleased]` for user-visible changes
- [ ] No secrets, credentials, or `.env` files committed

## Live Demo

To verify end-to-end behavior locally:

```powershell
.\scripts\demo.ps1              # Full 9-act demo (~80s)
.\scripts\demo.ps1 -SkipStackStart   # Stack already running
```

See [docs/demo/demo-script.md](docs/demo/demo-script.md) for the narrative.

## Release Tags

| Tag | Milestone |
|-----|-----------|
| `v0.1.0` | Phase 1 — core platform |
| `v0.2.0` | Phase 2 — gRPC, topics, retry/DLQ/replay, Helm, Terraform |
| `v0.3.0` | Demo system — Galactic Commerce |
| `v0.4.0` | CI/CD pipeline |
| `v1.0.0` | Documentation and portfolio release |

## Questions

Open a [GitHub issue](https://github.com/yucheng1554439/event-flow/issues) for bugs, feature requests, or design discussions.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
