# unicode

[![CI](https://github.com/SCKelemen/unicode/workflows/CI/badge.svg)](https://github.com/SCKelemen/unicode/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/SCKelemen/unicode)](https://goreportcard.com/report/github.com/SCKelemen/unicode)

Implementations of various Unicode® Standard Annexes in Go.

This repository provides Go packages for Unicode text processing algorithms, organized by UAX (Unicode Standard Annex) specification.

## Design Philosophy

These implementations focus on practical text layout and rendering needs:
- Simple, focused APIs
- Minimal dependencies (standard library only)
- Performance-conscious
- Well-tested
- Layout-engine agnostic
- Full conformance with Unicode standards

## Unicode Version

This repository implements **Unicode 17.0.0** (September 2024).

### Why Not Use Go's Standard Library?

Go's `unicode` package (as of Go 1.23) provides Unicode 15.0.0 data. While it includes some properties we need, it is missing many specialized properties required for text layout and rendering.

**Design Decision**: We implement all related properties within each specification package rather than mixing standard library and custom implementations. This ensures:

1. **Consistency**: All properties from a specification come from one authoritative source
2. **Completeness**: Unicode 17.0.0 support with the latest emoji and text handling
3. **Maintainability**: Single source of truth for each Unicode specification
4. **Testability**: 100% conformance against official Unicode 17.0.0 test files

When Go's `unicode` package updates to Unicode 17.0.0, we will continue maintaining our implementations to provide the specialized properties not available in the standard library.

## Installation

```bash
go get github.com/SCKelemen/unicode
```

## References

### Metastandards
- [UTR #33: Unicode Conformance Model](https://www.unicode.org/reports/tr33/) - Defines conformance requirements for Unicode Standard implementations
- [UAX #41: Common References for Unicode Standard Annexes](https://www.unicode.org/reports/tr41/) - Common definitions and references used across Unicode Standard Annexes

### Standards
- [Unicode Standard Annexes](https://www.unicode.org/reports/)
- [Unicode Character Database](https://www.unicode.org/Public/17.0.0/ucd/) - Character property data files

## License

MIT
