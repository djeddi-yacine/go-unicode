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

## Packages

### UAX #11: East Asian Width (`uax11`)

[![Go Reference](https://pkg.go.dev/badge/github.com/SCKelemen/unicode/uax11.svg)](https://pkg.go.dev/github.com/SCKelemen/unicode/uax11)

Implementation of [UAX #11 (East Asian Width)](https://www.unicode.org/reports/tr11/) for determining character display width in East Asian typography contexts.

**Key Features:**
- Character width classification (Fullwidth, Halfwidth, Wide, Narrow, Ambiguous, Neutral)
- Context-aware width resolution for Ambiguous characters
- Display width calculation for strings
- Terminal emulator support
- Text alignment and truncation utilities

**Quick Example:**
```go
import "github.com/SCKelemen/unicode/uax11"

// Check character width
width := uax11.LookupWidth('中')  // Returns Wide

// Calculate display width
displayWidth := uax11.StringWidth("Hello世界", uax11.ContextNarrow)  // Returns 9
```

[Full Documentation →](./uax11/README.md)

### UTS #51: Unicode Emoji (`uts51`)

[![Go Reference](https://pkg.go.dev/badge/github.com/SCKelemen/unicode/uts51.svg)](https://pkg.go.dev/github.com/SCKelemen/unicode/uts51)

Implementation of [UTS #51 (Unicode Emoji)](https://www.unicode.org/reports/tr51/) for complete emoji support in terminals, text editors, and layout engines.

**Key Features:**
- Six emoji properties (Emoji, Emoji_Presentation, Emoji_Modifier, Emoji_Modifier_Base, Emoji_Component, Extended_Pictographic)
- Terminal width calculation for emoji (integrates with UAX #11)
- Emoji sequence validation (keycap, tag, modifier, flag, ZWJ sequences)
- Presentation control (text vs emoji mode)
- 100% conformance (5,223/5,223 test cases passing)

**Quick Example:**
```go
import "github.com/SCKelemen/unicode/uts51"

// Check if character is emoji
if uts51.IsEmoji('😀') {
    fmt.Println("Is emoji!")
}

// Calculate width for terminal rendering
width := uts51.EmojiWidth('😀')  // Returns 2 (like CJK characters)
```

[Full Documentation →](./uts51/README.md)

### UAX #50: Vertical Text Layout (`uax50`)

[![Go Reference](https://pkg.go.dev/badge/github.com/SCKelemen/unicode/uax50.svg)](https://pkg.go.dev/github.com/SCKelemen/unicode/uax50)

Implementation of [UAX #50 (Unicode Vertical Text Layout)](https://www.unicode.org/reports/tr50/) for determining character orientation in vertical text.

**Key Features:**
- Vertical orientation property (Rotated, Upright, Transformed Upright, Transformed Rotated)
- Character orientation detection for East Asian typography
- Font shaping support for vertical text
- Mixed-script vertical text handling

**Quick Example:**
```go
import "github.com/SCKelemen/unicode/uax50"

// Check character orientation
if uax50.IsUpright('中') {
    fmt.Println("Display CJK ideograph upright")
}

if uax50.IsRotated('A') {
    fmt.Println("Rotate Latin letter 90° clockwise")
}
```

[Full Documentation →](./uax50/README.md)

### UAX #9: Bidirectional Algorithm (`uax9`)

[![Go Reference](https://pkg.go.dev/badge/github.com/SCKelemen/unicode/uax9.svg)](https://pkg.go.dev/github.com/SCKelemen/unicode/uax9)

Implementation of [UAX #9 (Unicode Bidirectional Algorithm)](https://www.unicode.org/reports/tr9/) for proper display of text containing both LTR and RTL scripts.

**Key Features:**
- Bidirectional text reordering for mixed LTR/RTL scripts
- Explicit formatting characters support (LRE, RLE, LRO, RLO, PDF, LRI, RLI, FSI, PDI)
- Automatic base direction detection
- Bracket pair handling (N0 rule)
- Full isolating run sequences (BD13)
- 100% conformance (513,494/513,494 test cases passing)

**Quick Example:**
```go
import "github.com/SCKelemen/unicode/uax9"

// Reorder mixed LTR/RTL text
text := "Hello שלום world"
result := uax9.Reorder(text, uax9.DirectionLTR)

// Auto-detect paragraph direction
dir := uax9.GetParagraphDirection("שלום עולם")
```

[Full Documentation →](./uax9/README.md)

### UAX #14: Line Breaking Algorithm (`uax14`)

[![Go Reference](https://pkg.go.dev/badge/github.com/SCKelemen/unicode/uax14.svg)](https://pkg.go.dev/github.com/SCKelemen/unicode/uax14)

Implementation of [UAX #14 (Unicode Line Breaking Algorithm)](https://www.unicode.org/reports/tr14/) for finding valid line break opportunities in text.

**Dependencies:** UAX #11 (East Asian Width)

**Key Features:**
- Finds valid line break opportunities according to UAX #14
- Three hyphenation modes (none, manual, auto)
- CJK ideographic text support
- Mandatory breaks (newlines, paragraph separators)
- Punctuation and numeric sequence rules
- 100% conformance (19,338/19,338 test cases passing)

**Quick Example:**
```go
import "github.com/SCKelemen/unicode/uax14"

text := "Hello world! This is a test."
breaks := uax14.FindLineBreakOpportunities(text, uax14.HyphensManual)

// Use break positions to wrap text
for i := 1; i < len(breaks); i++ {
    segment := text[breaks[i-1]:breaks[i]]
    fmt.Printf("%q\n", segment)
}
```

[Full Documentation →](./uax14/README.md)

### UAX #29: Text Segmentation (`uax29`)

[![Go Reference](https://pkg.go.dev/badge/github.com/SCKelemen/unicode/uax29.svg)](https://pkg.go.dev/github.com/SCKelemen/unicode/uax29)

Implementation of [UAX #29 (Unicode Text Segmentation)](https://www.unicode.org/reports/tr29/) for breaking text into grapheme clusters, words, and sentences.

**Dependencies:** UTS #51 (Unicode Emoji)

**Key Features:**
- Grapheme cluster boundaries (user-perceived characters)
- Word boundaries (text selection, cursor movement)
- Sentence boundaries (text processing)
- Emoji sequence handling with ZWJ
- Regional indicator sequences (flag emojis)
- Indic conjunct sequences
- 100% conformance (3,222/3,222 test cases passing)

**Quick Example:**
```go
import "github.com/SCKelemen/unicode/uax29"

// Break text into grapheme clusters
text := "👨‍👩‍👧‍👦 Hello"
clusters := uax29.Graphemes(text)

// Find word boundaries
words := uax29.Words("Hello, world!")

// Segment sentences
sentences := uax29.Sentences("Hello Dr. Smith. How are you?")
```

[Full Documentation →](./uax29/README.md)

## References

### Metastandards
- [UTR #33: Unicode Conformance Model](https://www.unicode.org/reports/tr33/) - Defines conformance requirements for Unicode Standard implementations
- [UAX #41: Common References for Unicode Standard Annexes](https://www.unicode.org/reports/tr41/) - Common definitions and references used across Unicode Standard Annexes

### Standards
- [Unicode Standard Annexes](https://www.unicode.org/reports/)
- [Unicode Character Database](https://www.unicode.org/Public/17.0.0/ucd/) - Character property data files

## License

BearWare 1.0 (MIT Compatible)
