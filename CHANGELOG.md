# Changelog

## [v6.2.0] - 2026-05-20

**Module path migration: `github.com/SCKelemen/unicode` → `github.com/SCKelemen/unicode/v6`.**

### Fixed

- **Module path now declares major version suffix.** Tags `v2.0.0` through `v6.1.0` were unreachable via `go get` because the `go.mod` declared `module github.com/SCKelemen/unicode` without the required `/v6` suffix. Go's module proxy refuses to serve v2+ tags at a path with no major-version suffix, so the v6 line was effectively unusable by downstream consumers.

  This release fixes the path declaration:

  ```go
  module github.com/SCKelemen/unicode/v6
  ```

  All internal imports between subpackages (`uax9`, `uax11`, `uax14`, `uax24`, `uax29`, `uax31`, `uax50`, `uts15`, `uts39`, `uts51`) have been rewritten to use the `/v6` path.

### Migration for downstream consumers

```bash
go get github.com/SCKelemen/unicode/v6@v6.2.0
```

Update imports:

```go
import "github.com/SCKelemen/unicode/v6/uax29"
```

Source-level API is unchanged from `v6.1.0`. This is purely a packaging fix.

### Historical tags

`v6.0.0` and `v6.1.0` git tags remain in the repository as historical artifacts of the v6 development line. They are not consumable via `go get` due to the path mismatch and will not be re-tagged (re-tagging would require destructive force-push, which is not policy for this repo). Use `v6.2.0` or later.

The `v1.x` line remains available at `github.com/SCKelemen/unicode` for consumers who have not yet migrated. `v1.1.1` is the most recent v1 release.
