----------------------
# name: master_agent
# repo: hack
# description: `hack` is a Go-based CLI for environment management, production workflows, and other utility commands for hackathon operations. It is intended to grow over time as the command surface expands.
# owner: anishalle
----------------------

# Repository Overview
`hack` is a small Cobra-based command-line application written in Go. The current repository is a scaffold, not a finished product, so agents should expect the command tree, flags, and supporting workflows to evolve.

## What Is In This Repo
- `main.go`: application entry point; delegates execution to the `cmd` package.
- `cmd/root.go`: defines the root `hack` command and program startup behavior.
- `cmd/auth.go`: defines the `auth` subcommand stub.
- `Makefile`: currently provides a basic `make` target that builds the binary as `./hack`.
- `go.mod` and `go.sum`: Go module definition and dependency lockfile.

## Current Architecture
- Go application using `cobra` for CLI command structure.
- UI-oriented dependencies are already present for future interactive flows: `bubbletea`, `bubbles`, and `lipgloss`.
- The project is currently command-centric; there is no documented config layer, persistence layer, or external service integration yet.

# General Agent Requirements
## Core Expectations
- Prefer small, focused changes that match the current repository shape.
- Keep the CLI usable from the terminal first; do not introduce abstraction layers unless they solve a concrete problem.
- Preserve existing command behavior unless the task explicitly requires a change.
- Use the repository's current patterns and dependencies before adding new ones.

## Code Quality
- Keep Go code idiomatic and straightforward.
- Use clear command names, help text, and error handling.
- Update or add tests when behavior changes, even if the current repo has no test suite yet.
- Avoid speculative features or dead code.

## Documentation Expectations
- Keep command descriptions accurate and concise.
- Update this file when the repo structure or primary workflows change.
- If a new command, config file, or subsystem is added, document it here in plain language.

## Working Notes For Agents
- Treat `hack` as the binary name unless the repo changes that convention.
- Assume the repository may remain a scaffold for some time; many commands may still be placeholders.
- When implementing new functionality, make the smallest coherent change that unblocks the next useful command or workflow.
