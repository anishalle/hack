# hack

HackUTD's internal CLI for managing environments, deployments, databases, and authentication across all projects.

Inspired by the GitHub CLI — interactive TUIs when you don't provide arguments, direct commands when you do.

## Quick Start

```bash
# Install
go install github.com/anishalle/hack/cmd/hack@latest

# Or build from source
make build
./bin/hack

# Login with your Google account
hack login

# Initialize a project
hack project init

# Pull environment variables
hack env pull dev

# Deploy to production
hack deploy up prod
```

## Commands

```
hack                              Interactive dashboard
hack login                        Google OAuth authentication
hack logout                       Clear credentials
hack whoami                       Show current user and project

hack project init                 Create hackfile.yaml
hack project list                 List accessible projects
hack project switch               Switch active project
hack project info                 Show project details

hack env                          Interactive env manager
hack env pull <env>               Pull env vars to .env file
hack env push <env>               Push .env to Secret Manager
hack env diff <env1> <env2>       Compare environments
hack env list <env>               List variable keys
hack env set <env> KEY=VALUE      Set a variable
hack env unset <env> KEY          Remove a variable
hack env edit <env>               Edit in $EDITOR
hack env history <env>            Change audit log

hack deploy                       Deployment dashboard
hack deploy up <env>              Deploy to environment
hack deploy status <env>          Service status
hack deploy logs <env>            Stream logs
hack deploy rollback <env>        Rollback deployment
hack deploy restart <env>         Restart services

hack db                           Database dashboard
hack db connect <env>             Open database connection
hack db status <env>              Database health
hack db migrate <env>             Run migrations
hack db backup <env>              Create backup
hack db branches <env>            Neon branch management

hack auth                         Auth provider dashboard
hack auth users <env>             List auth users
hack auth config <env>            Provider configuration
hack auth sessions <env>          Active sessions

hack admin                        Admin panel
hack admin users                  Manage team members
hack admin users add <email>      Invite user
hack admin users remove <email>   Remove user
hack admin roles                  View roles
hack admin roles assign <u> <r>   Assign role
hack admin audit                  View audit log
```

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

### hackfile.yaml

Each project has a `hackfile.yaml` that defines environments and providers:

```yaml
project: hackutd
version: "1"

environments:
  dev:
    deploy:
      provider: cloud-run
      project: hackutd-gcp
      region: us-central1
      service: hackutd-api-dev
    db:
      provider: neon
      project_id: aged-frost-12345
      branch: dev
    auth:
      provider: supertokens
      connection_uri: https://dev-xxxxx.supertokens.io

  prod:
    deploy:
      provider: cloud-run
      project: hackutd-gcp
      region: us-central1
      service: hackutd-api-prod
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

### RBAC

Permissions are granular and scoped per-project:

| Role | Permissions |
|------|-----------|
| **viewer** | `*:read` |
| **developer** | env:read, deploy:read, db:read, db:connect, auth:read |
| **deployer** | developer + deploy:write, deploy:restart, deploy:rollback |
| **admin** | all permissions |
| **owner** | all permissions + admin:* |

### Supported Providers

**Deployment:** Cloud Run, Compute Engine, App Engine

**Database:** Neon DB, Supabase, PlanetScale, Cloud SQL

**Authentication:** SuperTokens, Firebase Auth, Supabase Auth

**Secrets:** Google Secret Manager

## Backend Setup

The backend API runs on Cloud Run and requires:

- Google Cloud project with Firestore and Secret Manager enabled
- OAuth 2.0 client credentials
- JWT secret for token signing

```bash
# Set environment variables
export GCP_PROJECT=your-gcp-project
export GOOGLE_CLIENT_ID=your-client-id
export GOOGLE_CLIENT_SECRET=your-client-secret
export HACK_JWT_SECRET=your-jwt-secret

# Run locally
make run-server

# Build and deploy to Cloud Run
make server
docker build -t hack-server -f server/Dockerfile .
```

## Development

```bash
# Build CLI
make build

# Build server
make server

# Run tests
make test

# Format code
make fmt

# Install locally
make install
```

## Shell Completions

```bash
# Bash
hack completion bash > /etc/bash_completion.d/hack

# Zsh
hack completion zsh > "${fpath[1]}/_hack"

# Fish
hack completion fish > ~/.config/fish/completions/hack.fish
```
