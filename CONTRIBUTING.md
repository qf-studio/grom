# Contributing to grom

grom is open source under the [MIT License](LICENSE). Contributions are welcome and appreciated. This guide covers how to set up a development environment, submit changes, and follow project conventions.

## Development Setup

### Prerequisites

- **Go 1.25+** — [go.dev/dl](https://go.dev/dl/)
- **Make** — Build automation
- **Prometheus** (optional) — Any `/api/v1/query`-compatible endpoint for live dashboards. The demo gallery runs without one.

### Clone and Build

```bash
git clone https://github.com/qf-studio/grom.git
cd grom
make build
```

The binary is output to `bin/grom`.

### Run Without Prometheus

```bash
make demo                       # built-in demo gallery
./bin/grom demo --theme tokyo-night
```

### Run Tests

```bash
make test
```

### Lint

```bash
make lint
```

### Format

```bash
make fmt
```

## How to Contribute

### Report Bugs

Open a [GitHub Issue](https://github.com/qf-studio/grom/issues/new) with:
- Steps to reproduce
- Expected vs actual behavior
- grom version (`grom version`)
- OS, architecture, and terminal emulator (rendering bugs are often terminal-specific)

### Suggest Features

Open a GitHub Issue with the `enhancement` label. Describe the use case and why it matters. Feature discussions happen in issues before implementation starts.

### Submit Code

1. **Open an issue first** for non-trivial changes. This avoids duplicated effort and ensures alignment on approach.
2. Fork the repository
3. Create a feature branch from `main`
4. Make your changes
5. Run `make test && make lint` — both must pass
6. Submit a pull request

### Improve Documentation

Documentation lives in:
- `README.md` — Project overview and usage
- `examples/` — Example dashboard configs

PRs for documentation improvements follow the same process as code changes.

## Code Standards

### Go Conventions

- **Formatting**: `gofmt` (enforced by CI)
- **Vetting**: `go vet` (enforced by CI)
- **Linting**: `golangci-lint` (enforced by CI)
- **Testing**: Table-driven tests are the standard pattern
- **Dependencies**: Standard library first. The sanctioned UI dependency is the Charm stack (Bubble Tea / Lipgloss). Don't add dependencies casually.
- **Errors**: Return errors as values, wrap with `fmt.Errorf("...: %w", err)`. No panics in library code.

### Terminal Rendering

- All column/width math goes through `pkg/tui/render` (ANSI-aware). Never use `len()` on a styled string — visual width is `lipgloss.Width(s)`.
- Pull colors from `theme.Theme` tokens. Never hardcode hex values in render code.
- Render primitives get table-driven tests that include ANSI cases. Full panels get golden-output tests with fixtures under `testdata/`.

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): description

feat(render): ANSI-aware truncation
fix(prom): handle empty range-query result
refactor(grid): extract panel placement
test(widget): add braille chart golden tests
docs(readme): update installation instructions
chore(ci): upgrade Go version in workflow
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

### Tests and Prometheus

Unit tests never talk to a live Prometheus. Use recorded responses under `testdata/` instead.

## Pull Request Process

1. **CI must pass** — Tests, vetting, linting, and formatting are checked automatically
2. **One approval required** — A maintainer reviews all PRs
3. **Keep PRs focused** — One logical change per PR. Split large changes into smaller PRs.
4. **Update tests** — New features need tests. Bug fixes need regression tests.
5. **Update docs** — If your change affects user-facing behavior, update the README or examples.

## Project Structure

```
grom/
├── cmd/grom/            # CLI entrypoint
├── internal/
│   ├── app/             # Bubble Tea application model
│   ├── config/          # Dashboard config loading (YAML)
│   ├── datasource/      # Datasource interface
│   │   └── prom/        # Prometheus implementation
│   ├── grafana/         # Grafana dashboard JSON model + import
│   └── grid/            # Terminal grid layout
├── pkg/tui/
│   ├── render/          # ANSI-aware width/truncation primitives
│   ├── theme/           # Color themes and tokens
│   └── widget/          # Charts, meters, stats
├── examples/            # Example dashboard configs
└── docs/                # Demo recording (vhs tape + gif)
```

## License

grom is licensed under the [MIT License](LICENSE). By contributing, you agree that your contributions will be licensed under the same terms.

## Getting Help

- **GitHub Issues** — Bug reports and feature requests
- **Email** — hello@quantflow.studio

Thank you for contributing.
