package uax14

import "unicode"

// isCombiningMark checks if a rune is a combining mark (Mn or Mc).
// Used for LB9 rule: SA characters that are combining marks should be treated as CM.
func isCombiningMark(r rune) bool {
	return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r)
}

// QuoteContext tracks an opening quote for pairing.
type QuoteContext struct {
	pos   int16      // Position of opening quote (int16 saves memory, supports texts up to 32K runes)
	class BreakClass // ClassOP or ClassQU_Pf
	rune  rune       // Actual quote character
}

// LineBreakEnvironment tracks context state during forward pass.
// Eliminates backward scanning by maintaining state as we scan left-to-right.
// Pre-allocated at context creation, zero heap allocations during scanning.
type LineBreakEnvironment struct {
	// Quote tracking for LB19 (German quotes „..."）
	quoteStack         [8]QuoteContext // Stack of opening quotes (max 8 nesting levels)
	quoteTop           uint8           // Stack pointer (0 = empty)
	lastClosedQuote    int16           // Position of last closed German quote (-1 if none)
	lastClosedIsGerman bool            // Was the last closed quote a German quote?

	// Regional Indicator tracking for LB30a
	riCount    uint8 // Count of consecutive RIs seen
	riStartPos int16 // Position where RI sequence started

	// Hebrew hyphen tracking for LB21a
	lastHLPos int16 // Position of last HL (Hebrew Letter), -1 if none

	// Bracket/parenthesis depth tracking
	parenDepth uint8 // Nesting depth for balanced parentheses

	// Indic script state for Aksara rules (LB25, LB26, LB27)
	inAksaraSequence bool
	aksaraStartPos   int16

	// Space tracking for LB18
	lastNonSpacePos   int16      // Position of last non-SP character
	lastNonSpaceClass BreakClass // Class of last non-SP character
}

// LineBreakContext manages state for line break opportunity detection.
// It provides a clean abstraction over the text and classification data,
// making rule implementation straightforward and maintainable.
type LineBreakContext struct {
	// Input data (immutable)
	text    string
	runes   []rune
	classes []BreakClass
	hyphens Hyphens

	// Position tracking
	pos int

	// Cached lookups (updated on Slide())
	prevClass BreakClass
	currClass BreakClass
	nextClass BreakClass

	// State tracking for special rules
	lastNonSpaceClass BreakClass // For LB14 and other space-sensitive rules

	// Byte position tracking for output
	bytePositions []int

	// Environment for streaming parser (forward-only, zero backward scanning)
	env LineBreakEnvironment
}

// NewLineBreakContext creates a context for line break opportunity detection.
// It pre-classifies all runes for efficiency.
func NewLineBreakContext(text string, hyphens Hyphens) *LineBreakContext {
	if text == "" {
		return &LineBreakContext{
			text:    text,
			runes:   []rune{},
			classes: []BreakClass{},
			hyphens: hyphens,
			pos:     -1,
		}
	}

	runes := []rune(text)
	n := len(runes)

	// Pre-classify all runes
	classes := make([]BreakClass, n)
	for i, r := range runes {
		classes[i] = getBreakClass(r)
	}

	// Pre-compute byte positions for all rune boundaries
	bytePositions := make([]int, n+1)
	bytePositions[0] = 0
	bytePos := 0
	for i, r := range runes {
		bytePos += len(string(r))
		bytePositions[i+1] = bytePos
	}

	ctx := &LineBreakContext{
		text:          text,
		runes:         runes,
		classes:       classes,
		hyphens:       hyphens,
		pos:           0,
		bytePositions: bytePositions,
	}

	// Initialize cached classes and apply LB10 to first character
	ctx.updateCache()

	// LB10: Treat CM or ZWJ at start of text as AL
	if isClassOrVariant(ctx.prevClass, ClassCM) || ctx.prevClass == ClassZWJ {
		ctx.prevClass = ClassAL
	}
	ctx.lastNonSpaceClass = ctx.prevClass

	// Initialize environment for streaming parser
	ctx.env.lastHLPos = -1
	ctx.env.lastNonSpacePos = 0
	ctx.env.lastNonSpaceClass = ctx.prevClass
	ctx.env.riStartPos = -1
	ctx.env.lastClosedQuote = -1

	// Update environment for position 0
	ctx.updateEnvironment()

	return ctx
}

// Slide advances to the next position and returns true if there are more positions to check.
// Returns false when the end of the text is reached.
func (c *LineBreakContext) Slide() bool {
	c.pos++
	if c.pos >= len(c.runes) {
		return false
	}
	c.updateCache()
	c.updateEnvironment()
	return true
}

// updateCache updates the cached prev/curr/next class values for the current position.
func (c *LineBreakContext) updateCache() {
	if c.pos > 0 {
		c.prevClass = c.classes[c.pos-1]
	} else {
		c.prevClass = ClassAL // Default for position 0
	}

	if c.pos < len(c.classes) {
		c.currClass = c.classes[c.pos]

		// LB9: SA characters that are combining marks (Mn, Mc) should be treated as CM
		if c.currClass == ClassSA {
			// Check if it's a combining mark
			if isCombiningMark(c.runes[c.pos]) {
				c.currClass = ClassCM
			}
		}
	} else {
		c.currClass = ClassAL
	}

	if c.pos+1 < len(c.classes) {
		c.nextClass = c.classes[c.pos+1]
	} else {
		c.nextClass = ClassAL
	}
}

// updateEnvironment updates the environment state for the current position.
// Called after updateCache() during Slide() to maintain forward-only state.
func (c *LineBreakContext) updateEnvironment() {
	curr := c.currClass
	pos16 := int16(c.pos)

	// Update last non-space position
	if curr != ClassSP {
		c.env.lastNonSpacePos = pos16
		c.env.lastNonSpaceClass = curr
	}

	// Track Hebrew letters for LB21a
	if curr == ClassHL {
		c.env.lastHLPos = pos16
	}

	// Track Regional Indicators for LB30a
	// LB9: Treat X (CM | ZWJ)* as if it were X - skip CM/ZWJ when counting RIs
	if curr == ClassRI {
		if c.env.riCount == 0 {
			c.env.riStartPos = pos16
		}
		c.env.riCount++
	} else if !isClassOrVariant(curr, ClassCM) && curr != ClassZWJ {
		// Reset RI count when we hit non-RI (but skip CM/ZWJ per LB9)
		c.env.riCount = 0
		c.env.riStartPos = -1
	}

	// Track opening quotes for LB19 (German quotes)
	// German opening quotes: U+201E („) and U+201A (‚) are ClassOP
	if curr == ClassOP {
		r := c.runes[c.pos]
		if r == '\u201E' || r == '\u201A' {
			// Push German opening quote to stack
			if c.env.quoteTop < 8 {
				c.env.quoteStack[c.env.quoteTop] = QuoteContext{
					pos:   pos16,
					class: curr,
					rune:  r,
				}
				c.env.quoteTop++
			}
		}
	}

	// Track closing quotes for LB19 (German quotes)
	// German closing quotes: U+201C (") and U+2018 (') are ClassQU_Pi
	if curr == ClassQU_Pi {
		r := c.runes[c.pos]
		if r == '\u201C' || r == '\u2018' {
			// Pop matching German opening quote from stack
			if c.env.quoteTop > 0 {
				// Check if there's a matching German opening quote
				top := c.env.quoteStack[c.env.quoteTop-1]
				if top.rune == '\u201E' || top.rune == '\u201A' {
					c.env.quoteTop--
					c.env.lastClosedQuote = pos16
					c.env.lastClosedIsGerman = true
				}
			}
		}
	}

	// Track parenthesis depth
	if curr == ClassOP {
		c.env.parenDepth++
	} else if curr == ClassCP {
		if c.env.parenDepth > 0 {
			c.env.parenDepth--
		}
	}

	// Track Aksara sequences for Indic scripts
	if curr == ClassAK || curr == ClassAS {
		if !c.env.inAksaraSequence {
			c.env.inAksaraSequence = true
			c.env.aksaraStartPos = pos16
		}
	} else if curr == ClassVF {
		// Virama Final ends Aksara sequence
		c.env.inAksaraSequence = false
	}
}

// BytePos returns the byte offset for the current position.
func (c *LineBreakContext) BytePos() int {
	if c.pos < 0 || c.pos >= len(c.bytePositions) {
		return len(c.text)
	}
	return c.bytePositions[c.pos]
}

// Pos returns the current rune position.
//
//go:inline
func (c *LineBreakContext) Pos() int {
	return c.pos
}

// Prev returns the line break class of the previous character.
//
//go:inline
func (c *LineBreakContext) Prev() BreakClass {
	return c.prevClass
}

// Curr returns the line break class of the current character.
//
//go:inline
func (c *LineBreakContext) Curr() BreakClass {
	return c.currClass
}

// Next returns the line break class of the next character.
//
//go:inline
func (c *LineBreakContext) Next() BreakClass {
	return c.nextClass
}

// Env returns a pointer to the environment for streaming parser state access.
//
//go:inline
func (c *LineBreakContext) Env() *LineBreakEnvironment {
	return &c.env
}

// Rune returns the rune at the current position.
//
//go:inline
func (c *LineBreakContext) Rune() rune {
	if c.pos < 0 || c.pos >= len(c.runes) {
		return 0
	}
	return c.runes[c.pos]
}

// RuneAt returns the rune at the specified position.
//
//go:inline
func (c *LineBreakContext) RuneAt(pos int) rune {
	if pos < 0 || pos >= len(c.runes) {
		return 0
	}
	return c.runes[pos]
}

// ClassAt returns the line break class at the specified position.
//
//go:inline
func (c *LineBreakContext) ClassAt(pos int) BreakClass {
	if pos < 0 || pos >= len(c.classes) {
		return ClassAL // Default
	}
	return c.classes[pos]
}

// LastNonSpace returns the last non-SP class seen.
//
//go:inline
func (c *LineBreakContext) LastNonSpace() BreakClass {
	return c.lastNonSpaceClass
}

// UpdatePrevClass updates the previous class (for state transitions after breaks).
//
//go:inline
func (c *LineBreakContext) UpdatePrevClass(class BreakClass) {
	c.prevClass = class
	if class != ClassSP {
		c.lastNonSpaceClass = class
	}
}

// UpdateLastNonSpace updates the last non-space class tracker.
//
//go:inline
func (c *LineBreakContext) UpdateLastNonSpace(class BreakClass) {
	c.lastNonSpaceClass = class
}

// Len returns the total number of runes.
//
//go:inline
func (c *LineBreakContext) Len() int {
	return len(c.runes)
}

// Hyphens returns the hyphenation mode.
//
//go:inline
func (c *LineBreakContext) Hyphens() Hyphens {
	return c.hyphens
}

// LookBack looks backward from the current position for a specific pattern.
// Returns the position if found, -1 otherwise.
func (c *LineBreakContext) LookBack(distance int) int {
	targetPos := c.pos - distance
	if targetPos < 0 {
		return -1
	}
	return targetPos
}

// GetPairTableAction looks up the break action from the pair table.
func (c *LineBreakContext) GetPairTableAction(prev, curr BreakClass) BreakAction {
	return getBreakAction(prev, curr)
}

// SkipBackward skips over specified classes backward from startIdx, returning the index
// of the first non-skipped class (or -1 if start of text reached).
// Commonly used to skip CM/ZWJ per LB9: treat X (CM | ZWJ)* as X
func (c *LineBreakContext) SkipBackward(startIdx int, skipClasses ...BreakClass) int {
	idx := startIdx
	for idx >= 0 {
		class := c.ClassAt(idx)
		shouldSkip := false
		for _, skipClass := range skipClasses {
			if class == skipClass || isClassOrVariant(class, skipClass) {
				shouldSkip = true
				break
			}
		}
		if !shouldSkip {
			return idx
		}
		idx--
	}
	return -1
}

// SkipForward skips over specified classes forward from startIdx, returning the index
// of the first non-skipped class (or c.Len() if end of text reached).
func (c *LineBreakContext) SkipForward(startIdx int, skipClasses ...BreakClass) int {
	idx := startIdx
	for idx < c.Len() {
		class := c.ClassAt(idx)
		shouldSkip := false
		for _, skipClass := range skipClasses {
			if class == skipClass || isClassOrVariant(class, skipClass) {
				shouldSkip = true
				break
			}
		}
		if !shouldSkip {
			return idx
		}
		idx++
	}
	return c.Len()
}

// FindForward searches forward from startIdx until it finds one of the target classes.
// Returns the index where found, or -1 if not found within maxDistance.
// Optionally skips over specified classes (like CM/ZWJ).
func (c *LineBreakContext) FindForward(startIdx int, maxDistance int, targetClasses []BreakClass, skipClasses ...BreakClass) int {
	idx := startIdx
	limit := c.Len()
	if maxDistance > 0 && startIdx+maxDistance < limit {
		limit = startIdx + maxDistance
	}

	for idx < limit {
		class := c.ClassAt(idx)

		// Check if this is a skip class
		shouldSkip := false
		for _, skipClass := range skipClasses {
			if class == skipClass || isClassOrVariant(class, skipClass) {
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			idx++
			continue
		}

		// Check if this is a target class
		for _, target := range targetClasses {
			if class == target || isClassOrVariant(class, target) {
				return idx
			}
		}

		// Not a target, stop searching
		return -1
	}
	return -1
}

// FindBackward searches backward from startIdx until it finds one of the target classes.
// Returns the index where found, or -1 if not found.
func (c *LineBreakContext) FindBackward(startIdx int, targetClasses []BreakClass, skipClasses ...BreakClass) int {
	idx := startIdx

	for idx >= 0 {
		class := c.ClassAt(idx)

		// Check if this is a skip class
		shouldSkip := false
		for _, skipClass := range skipClasses {
			if class == skipClass || isClassOrVariant(class, skipClass) {
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			idx--
			continue
		}

		// Check if this is a target class
		for _, target := range targetClasses {
			if class == target || isClassOrVariant(class, target) {
				return idx
			}
		}

		// Not a target, stop searching
		return -1
	}
	return -1
}

// MatchSequence checks if a sequence of classes matches forward from startIdx.
// Returns true if the entire sequence matches.
func (c *LineBreakContext) MatchSequence(startIdx int, sequence ...BreakClass) bool {
	for i, class := range sequence {
		idx := startIdx + i
		if idx >= c.Len() {
			return false
		}
		actualClass := c.ClassAt(idx)
		if actualClass != class && !isClassOrVariant(actualClass, class) {
			return false
		}
	}
	return true
}
