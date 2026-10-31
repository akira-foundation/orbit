# Contributing to Orbit

Thanks for your interest in contributing.

## Bugs and feature requests

Open an issue at https://github.com/akira-foundation/orbit/issues. Include:

- What you expected to happen.
- What happened instead.
- A minimal reproduction.
- Versions: Orbit, Xcode and macOS.

## Working on a pull request

1. Create a branch from `develop` named `<type>/<issue-number>-<kebab-title>`, for example `feat/1-cli`.
2. Build with Xcode 27 on macOS 27. Add tests for the change.
3. Run `make lint && make swiftlint && make format-check && make test && make test-design && make build`
   before pushing. CI runs the same checks on every pull request.
4. Use Conventional Commits (`feat(agent): serve sites over HTTPS`). The changelog is generated from
   them via [git-cliff](https://git-cliff.org). Mark breaking changes with `!`. The allowed scopes are listed in
   `CLAUDE.md`.
5. Open the pull request against `develop`, or against the `milestone/*` or `release/*` branch the work
   belongs to. Keep the diff focused: refactors, feature work and dependency bumps belong in separate pull
   requests.

## Style

- swift-format and SwiftLint are the source of truth; `make format` rewrites the sources.
- `Tools/house-rules.sh` holds the rules no linter knows: only the app, `OrbitDesign` and its tests import
  SwiftUI or AppKit, no `Helpers/` or `Extensions/` folders, no view runs a subprocess.
- No comments that narrate the code. Names should carry the meaning.
- No drive-by refactors in feature pull requests.
- No emojis in code, copy, commit messages or pull request descriptions.

## License

By contributing, you agree that your contributions will be dual-licensed
under MIT and Apache-2.0, as described in [LICENSE-MIT](LICENSE-MIT) and
[LICENSE-APACHE](LICENSE-APACHE).
