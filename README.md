# hack

HackUTD's internal CLI for managing environments, deployments, databases, and authentication across all projects.

Inspired by the GitHub CLI — interactive TUIs when you don't provide arguments, direct commands when you do.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Backend Setup](#backend-setup)
- [Getting Started (From Scratch)](#getting-started-from-scratch)
- [Commands Reference](#commands-reference)
- [hackfile.yaml Configuration](#hackfileyaml-configuration)
- [Architecture](#architecture)
- [RBAC & Permissions](#rbac--permissions)
- [What Works Today](#what-works-today)
- [What Doesn't Work Yet](#what-doesnt-work-yet)
- [Development](#development)

---

## Prerequisites

Before you start, make sure you have:

| Dependency | Why |
|---|---|
| **Go 1.25+** | CLI and server are written in Go |
| **Google Cloud project** | Firestore (user/project data), Secret Manager (env vars), and OAuth |
| **Google OAuth 2.0 credentials** | CLI login uses device-code flow with Google |
| **Neon account** | Postgres database provider (configured per-environment) |
| **SuperTokens account** | Auth provider for your app's end-users (configured per-environment) |
| **Docker** (optional) | Only needed if deploying the backend server to Cloud Run |
| **psql** (optional) | For `hack db connect` to open interactive Postgres sessions |

### Google Cloud setup checklist

1. Create a GCP project (or use an existing one)
2. Enable the **Firestore** API (Native mode)
3. Enable the **Secret Manager** API
4. Create an **OAuth 2.0 Client ID** (type: Web application)
   - Note the Client ID and Client Secret — you'll need them for the backend
5. Make sure your Google account has IAM permissions for Firestore and Secret Manager on the project

### Neon setup checklist

1. Create a Neon account at [neon.tech](https://neon.tech)
2. Create a project — note the **project ID** (looks like `aged-frost-12345`)
3. Your project will have a `main` branch by default; create a `dev` branch for development

### SuperTokens setup checklist

1. Create a SuperTokens account at [supertokens.com](https://supertokens.com)
2. Create apps for each environment — note the **connection URIs** (e.g. `https://dev-xxxxx.supertokens.io`)

---

## Installation

```bash
# Install directly from source
go install github.com/anishalle/hack/cmd/hack@latest

# Or clone and build
git clone https://github.com/anishalle/hack.git
cd hack
make build
./bin/hack
```

After installing, verify it works:

```bash
hack version
```

---

## Backend Setup

The `hack` CLI talks to a backend API server. You need to run this server (locally for dev, or on Cloud Run for production).

### Environment variables

| Variable | Required | Description |
|---|---|---|
| `GCP_PROJECT` | Yes | Your Google Cloud project ID |
| `GOOGLE_CLIENT_ID` | Yes | OAuth 2.0 Client ID |
| `GOOGLE_CLIENT_SECRET` | Yes | OAuth 2.0 Client Secret |
| `HACK_JWT_SECRET` | Yes (prod) | Secret for signing JWT tokens. Defaults to `dev-secret-change-in-production` locally |
| `PORT` | No | HTTP port, defaults to `8080` |

### Run the backend locally

```bash
export GCP_PROJECT=your-gcp-project
export GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
export GOOGLE_CLIENT_SECRET=your-client-secret
export HACK_JWT_SECRET=some-strong-random-secret

make run-server
```

The server starts at `http://localhost:8080`. The CLI points here by default.

### Deploy the backend to Cloud Run

```bash
# Build the server binary
make server

# Build Docker image
docker build -t hack-server -f server/Dockerfile .

# Tag and push to Artifact Registry, then deploy to Cloud Run
# (use your own registry and project)
```

If you deploy the backend somewhere other than localhost, update the CLI's API URL:

```bash
# Edit ~/.hack/config.yaml
api:
  base_url: https://your-hack-server.run.app
```

---

## Getting Started (From Scratch)

Here's the complete flow for a brand-new user joining an existing team or starting a new project.

### Step 1: Install the CLI

```bash
go install github.com/anishalle/hack/cmd/hack@latest
```

### Step 2: Log in

```bash
hack login
```

This opens your browser for Google OAuth. A device code is displayed in the terminal — in headless environments (SSH, containers), enter the code at the URL shown.

After login, your credentials are stored at `~/.hack/config.yaml`.

### Step 3: Initialize your project

Navigate to your project's root directory and run:

```bash
hack project init
```

The interactive form asks for:
- **Project name** — a short identifier like `hackutd`
- **GCP project ID** — the Google Cloud project holding your secrets
- **Environments** — pick from dev, staging, prod (defaults to dev + prod)

This creates a `hackfile.yaml` in your project root. Open it and fill in the provider-specific fields:

```yaml
project: hackutd
version: "1"

environments:
  dev:
    deploy:
      provider: cloud-run
      project: hackutd-gcp
      region: us-central1
      service: hackutd-api-dev      # your Cloud Run service name
      dockerfile: ./Dockerfile
    db:
      provider: neon
      project_id: aged-frost-12345  # from Neon dashboard
      branch: dev                   # Neon branch name
    auth:
      provider: supertokens
      connection_uri: https://dev-xxxxx.supertokens.io  # from SuperTokens dashboard

  prod:
    deploy:
      provider: cloud-run
      project: hackutd-gcp
      region: us-central1
      service: hackutd-api-prod
      dockerfile: ./Dockerfile
    db:
      provider: neon
      project_id: aged-frost-12345
      branch: main
    auth:
      provider: supertokens
      connection_uri: https://prod-xxxxx.supertokens.io

secrets:
  provider: google-secret-manager
  project: hackutd-gcp
  prefix: hackutd
```

### Step 4: Register with the backend

```bash
hack project register
```

This tells the hack server about your project so it can manage permissions, env vars, and audit logs.

### Step 5: Start using it

```bash
# Set your first env variable
hack env set dev DATABASE_URL=postgres://user:pass@your-neon-host/dbname

# Pull all env vars to a local .env file
hack env pull dev

# See who you're logged in as
hack whoami

# View project details
hack project info
```

---

## Commands Reference

Every command has an interactive mode (run without arguments) and a direct mode (pass arguments).

### Authentication

| Command | Description |
|---|---|
| `hack login` | Google OAuth login via device-code flow |
| `hack logout` | Clear stored credentials |
| `hack whoami` | Show current user email and active project |

### Projects

| Command | Description |
|---|---|
| `hack project` | Interactive project switcher |
| `hack project init` | Create `hackfile.yaml` via guided form |
| `hack project list` | List projects you have access to |
| `hack project switch [name]` | Change active project context |
| `hack project info` | Show active project details and environments |
| `hack project register` | Register current project with the backend |

### Environment Variables

Env vars are stored in Google Secret Manager. Access is controlled by your project role.

| Command | Description |
|---|---|
| `hack env` | Interactive env var browser |
| `hack env pull <env>` | Download env vars to `.env.<env>` file |
| `hack env push <env>` | Upload local `.env.<env>` to Secret Manager |
| `hack env diff <env1> <env2>` | Side-by-side diff of two environments |
| `hack env list <env>` | List variable keys (values hidden by default) |
| `hack env set <env> KEY=VALUE` | Set one or more variables |
| `hack env unset <env> KEY` | Remove a variable |
| `hack env edit <env>` | Open in `$EDITOR` (defaults to vim) |
| `hack env history <env>` | View change audit log |

**Flags:**
- `hack env pull --output .env` — custom output file
- `hack env pull --force` — overwrite without confirmation
- `hack env push --file .env.local` — push from a specific file
- `hack env push --dry-run` — preview without applying
- `hack env list --values` — reveal values (hidden by default)

### Deployments

| Command | Description |
|---|---|
| `hack deploy` | Interactive deployment dashboard |
| `hack deploy up <env>` | Deploy to an environment |
| `hack deploy status <env>` | Check deployment status |
| `hack deploy logs <env>` | View deployment logs |
| `hack deploy rollback <env>` | Rollback to previous revision |
| `hack deploy restart <env>` | Restart running services |

**Flags:**
- `hack deploy up --tag v1.2.3` — deploy a specific image tag
- `hack deploy up --yes` — skip production confirmation prompt
- `hack deploy logs --follow` — stream logs in real-time
- `hack deploy logs --tail 100` — number of recent lines
- `hack deploy rollback --revision <rev>` — target a specific revision

**Note:** Deploying to `prod` or `production` triggers a confirmation prompt unless `--yes` is passed.

### Database (Neon)

| Command | Description |
|---|---|
| `hack db` | Interactive database dashboard |
| `hack db connect <env>` | Open psql session to the environment's database |
| `hack db status <env>` | Check database health |
| `hack db migrate <env>` | Run database migrations |
| `hack db backup <env>` | Create a database backup |
| `hack db branches <env>` | List Neon database branches |

**Flags:**
- `hack db connect --tool pgcli` — use pgcli instead of psql
- `hack db migrate --dry-run` — preview without executing

### Auth Provider (SuperTokens)

| Command | Description |
|---|---|
| `hack auth` | Interactive auth dashboard |
| `hack auth users <env>` | List users from your auth provider |
| `hack auth config <env>` | View auth provider configuration |
| `hack auth sessions <env>` | View active sessions |

**Flags:**
- `hack auth users --limit 50` — max users to display
- `hack auth users --search user@email.com` — search by email or ID

### Admin

Requires `admin` or `owner` role on the project.

| Command | Description |
|---|---|
| `hack admin` | Interactive admin panel |
| `hack admin users` | List project team members and roles |
| `hack admin users add <email>` | Invite a user to the project |
| `hack admin users remove <email>` | Remove a user (with confirmation) |
| `hack admin roles` | List available roles and their permissions |
| `hack admin roles assign <email> <role>` | Assign a role to a user |
| `hack admin audit` | View the project audit log |

**Flags:**
- `hack admin users add --role deployer` — set role on invite (default: developer)
- `hack admin audit --action env` — filter by action type
- `hack admin audit --user alice@example.com` — filter by user
- `hack admin audit --limit 50` — max entries

### Other

| Command | Description |
|---|---|
| `hack version` | Print CLI version |
| `hack` (no args) | Open interactive dashboard |

---

## hackfile.yaml Configuration

The `hackfile.yaml` lives in your project root and defines environments and providers.

### Full schema

```yaml
project: <string>          # Project identifier (must match what's registered with backend)
version: "1"               # Config version

environments:
  <env_name>:              # e.g. dev, staging, prod
    deploy:
      provider: cloud-run  # Deployment provider
      project: <string>    # GCP project ID
      region: <string>     # e.g. us-central1
      service: <string>    # Cloud Run service name
      dockerfile: <string> # Path to Dockerfile (e.g. ./Dockerfile)
    db:
      provider: neon       # Database provider
      project_id: <string> # Neon project ID
      branch: <string>     # Neon branch name (e.g. dev, main)
    auth:
      provider: supertokens       # Auth provider
      connection_uri: <string>    # SuperTokens connection URI
      project: <string>           # (optional) SuperTokens project name

secrets:
  provider: google-secret-manager  # Secrets backend
  project: <string>                # GCP project ID
  prefix: <string>                 # Prefix for secret names (usually = project name)
```

### User config (`~/.hack/config.yaml`)

Created automatically on first login. You can edit `api.base_url` to point to a non-local backend.

```yaml
active_project: hackutd
api:
  base_url: http://localhost:8080   # Change if backend is deployed elsewhere
auth:
  access_token: <jwt>
  refresh_token: <token>
  expires_at: <timestamp>
  email: you@example.com
projects:
  hackutd:
    path: /Users/you/projects/hackutd
    default_env: dev
```

---

## Architecture

```
CLI (hack binary)              Backend API (Cloud Run)
┌──────────────────┐           ┌──────────────────────┐
│  Cobra Commands  │──────────▶│  chi HTTP Router     │
│  Bubbletea TUIs  │  HTTP/    │  RBAC Middleware      │
│  Local Config    │  JWT      │  Firestore Store      │
└──────────────────┘           │  Provider Layer       │
                               └──────┬───────────────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    ▼                 ▼                  ▼
            Google Secret      Cloud Run /         Neon DB /
            Manager            Compute Engine      SuperTokens
```

- **CLI** is a single Go binary. It reads `hackfile.yaml` from the project directory and `~/.hack/config.yaml` for auth/state.
- **Backend** is a Go HTTP server using chi. It handles auth (Google OAuth device-code flow), stores data in Firestore, manages secrets via Google Secret Manager, and enforces RBAC.
- All communication is over HTTP with JWT bearer tokens.

---

## RBAC & Permissions

Roles are per-project. The person who registers a project becomes the `owner`.

| Role | Permissions |
|---|---|
| **viewer** | `env:read`, `deploy:read`, `db:read`, `auth:read` |
| **developer** | Everything in viewer + `db:connect` |
| **deployer** | Everything in developer + `deploy:write`, `deploy:restart`, `deploy:rollback` |
| **admin** | All permissions except owner-only operations |
| **owner** | All permissions including `admin:*` (user/role management) |

Assign roles via:

```bash
hack admin users add teammate@example.com --role deployer
hack admin roles assign teammate@example.com admin
```

---

## What Works Today

These features are fully wired end-to-end (CLI -> backend -> provider):

- **Authentication** — Google OAuth device-code login, JWT tokens (24h access, 30-day refresh), automatic token refresh
- **Project management** — create, register, list, switch, view info
- **Environment variables** — pull, push, set, unset, diff, list, edit in `$EDITOR`, all backed by Google Secret Manager
- **Admin** — list/add/remove users, assign roles, view audit log
- **Interactive TUIs** — dashboard, env browser, and interactive selectors for all commands when run without arguments

## What Doesn't Work Yet

These features exist in the CLI but are **not fully wired** on the backend:

| Feature | Status | Details |
|---|---|---|
| `hack deploy *` | **Stubs** | CLI sends requests, backend has handlers but they return placeholder responses. Cloud Run provider code exists but isn't called by the handlers. |
| `hack db connect` | **No route** | Backend has no `/db/*` routes. CLI calls will 404. Neon provider code exists but is unused. |
| `hack db status` | **No route** | Same — no backend route. |
| `hack db branches` | **No route** | Same. |
| `hack db backup` | **No route** | Same. |
| `hack db migrate` | **Stub** | CLI prints a success message without actually running migrations. |
| `hack auth users` | **No route** | Backend has no `/auth/*` routes. SuperTokens provider code exists but is unused. |
| `hack auth sessions` | **No route** | Same. |
| `hack deploy logs` | **Empty** | Handler exists but returns an empty array; Cloud Logging is not integrated. |
| `hack deploy logs --follow` | **Not implemented** | The `--follow` flag is accepted but ignored. |
| `hack env history --limit` | **Ignored** | The limit flag is parsed but not sent to the API. |
| Shell completions | **Not implemented** | The `hack completion` commands mentioned in the old README are not registered. |

### Providers that exist but aren't used

The codebase has provider implementations for:
- **Cloud Run** (`server/internal/provider/deploy/cloudrun.go`) — Deploy, Status, Rollback, Restart
- **Neon** (`server/internal/provider/db/neon.go`) — Connect, Status, Branches
- **SuperTokens** (`server/internal/provider/auth/supertokens.go`) — Users, Sessions

These will work once the backend handlers are wired up to call them.

---

## Development

```bash
# Build the CLI
make build

# Build the backend server
make server

# Run the CLI (without building)
make run

# Run the backend server (without building)
make run-server

# Run tests
make test

# Format code
make fmt

# Lint (requires golangci-lint)
make lint

# Install CLI to $GOPATH/bin
make install

# Clean build artifacts
make clean
```

### Project structure

```
hack/
├── cmd/hack/main.go              # CLI entrypoint
├── internal/
│   ├── api/                      # HTTP client for talking to backend
│   ├── cli/                      # All Cobra commands
│   ├── config/                   # User config + hackfile parsing
│   ├── models/                   # Shared data models
│   └── tui/                      # Bubbletea TUI components
├── server/
│   ├── cmd/server/main.go        # Backend server entrypoint
│   ├── Dockerfile
│   └── internal/
│       ├── handler/              # HTTP route handlers
│       ├── middleware/           # JWT auth, RBAC enforcement
│       ├── provider/            # Cloud Run, Neon, SuperTokens integrations
│       └── store/               # Firestore data layer
├── hackfile.yaml                 # Example project config
├── Makefile
└── go.mod
```
