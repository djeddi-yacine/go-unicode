# Streaming Parser Design

## Problem: Backward Scanning

Current implementation requires backward scanning for several rules:

1. **LB19 (German quotes)**: Scan back up to 30 positions to find opening quote
2. **LB30a (Regional Indicators)**: Count RI pairs backward
3. **LB21a (Hebrew + Hyphen)**: Look back for HL before HH
4. **Quote matching**: Track opening/closing quote pairs
5. **Bracket sequences**: Match opening/closing punctuation

Each backward scan is O(N) per character, making the algorithm O(N²) in worst case.

## Solution: Forward-Only Single-Pass Parser

Use a **LineBreakEnvironment** to track state during forward pass, eliminating backward scans.

## Architecture

### LineBreakEnvironment Structure

```go
// LineBreakEnvironment tracks context state during forward pass.
// Pre-allocated at start, updated as we scan forward, zero backward scanning.
type LineBreakEnvironment struct {
    // Quote tracking for LB19 (German quotes)
    quoteStack []QuoteContext  // Stack of opening quotes

    // Regional Indicator tracking for LB30a
    riCount uint8  // Count of consecutive RIs seen
    riStartPos int // Position where RI sequence started

    // Hebrew hyphen tracking for LB21a
    lastHLPos int  // Position of last HL (Hebrew Letter)

    // Bracket/parenthesis depth tracking
    parenDepth uint8   // Nesting depth for balanced parentheses
    bracketDepth uint8 // Nesting depth for balanced brackets

    // Indic script state for Aksara rules (LB25, LB26, LB27)
    inAksaraSequence bool
    aksaraStartPos int

    // Space tracking for LB18
    lastNonSpacePos int
    lastNonSpaceClass BreakClass
}

// QuoteContext tracks an opening quote for pairing.
type QuoteContext struct {
    pos   int         // Position of opening quote
    class BreakClass  // ClassOP or ClassQU_Pf
    rune  rune        // Actual quote character
}
```

### Key Optimizations

1. **Pre-allocated stacks**: Fixed-size arrays, no heap allocations
   - Quote stack: max 8 levels (deeply nested quotes are rare)
   - RI tracking: simple counter (max sequence ~10 flags)

2. **Forward-only updates**: Environment updated as we scan left-to-right
   - Opening quote? Push to stack
   - Closing quote? Pop and check match
   - Regional indicator? Increment counter or reset

3. **Zero backward scanning**: All lookups are O(1) from environment
   - "Is there an opening quote before this?" → Check stack top
   - "How many RIs before current position?" → Read riCount
   - "Was there an HL recently?" → Check lastHLPos

## Rule Adaptations

### LB19 (German Quotes) - Before

```go
// Current: Scan backward up to 30 positions
func ruleLB19_German(ctx *LineBreakContext) (bool, BreakDecision) {
    if ctx.Curr() != ClassQU_Pi {
        return false, BreakNo
    }

    // Scan backward looking for opening quote
    for checkIdx := ctx.Pos() - 3; checkIdx >= 0 && checkIdx > ctx.Pos()-30; checkIdx-- {
        // ... complex backward scanning logic ...
    }
}
```

### LB19 (German Quotes) - After

```go
// Streaming: Check environment stack (O(1))
func ruleLB19_German(ctx *LineBreakContext) (bool, BreakDecision) {
    if ctx.Curr() != ClassQU_Pi {
        return false, BreakNo
    }

    // Check if we have a matching opening quote in environment
    if env := ctx.Env(); len(env.quoteStack) > 0 {
        opening := env.quoteStack[len(env.quoteStack)-1]
        if opening.class == ClassOP {
            // Found matching pair: „...\" pattern
            return true, BreakNo
        }
    }

    return false, BreakNo
}
```

### LB30a (Regional Indicators) - Before

```go
// Current: Count RIs backward
func ruleLB30a(ctx *LineBreakContext) (bool, BreakDecision) {
    // ... scan backward counting RIs, skipping CM/ZWJ ...
}
```

### LB30a (Regional Indicators) - After

```go
// Streaming: Read counter from environment
func ruleLB30a(ctx *LineBreakContext) (bool, BreakDecision) {
    curr := ctx.Curr()
    if curr != ClassRI {
        return false, BreakNo
    }

    // Check RI count in environment (already tracked during forward scan)
    if ctx.Env().riCount % 2 == 1 {
        // Odd number of RIs: don't break
        return true, BreakNo
    }

    return false, BreakNo
}
```

## Implementation Strategy

### Phase 1: Add Environment Structure (Non-Breaking)

1. Add `LineBreakEnvironment` struct to context.go
2. Add `env *LineBreakEnvironment` field to LineBreakContext
3. Initialize environment in NewLineBreakContext()
4. Update Slide() to maintain environment state
5. Add Env() accessor method

**Changes are additive** - existing code still works.

### Phase 2: Port Rules to Use Environment

Port rules one category at a time:

1. **Quote rules** (LB19, LB15) - Use quoteStack
2. **RI rules** (LB30a) - Use riCount
3. **Hebrew rules** (LB21a) - Use lastHLPos
4. **Space rules** (LB18) - Use lastNonSpacePos (already tracked!)

**Test after each category** to maintain 100% conformance.

### Phase 3: Optimize Environment Updates

Once all rules use environment, optimize the update logic:

1. Inline environment updates in Slide()
2. Remove conditional branches where possible
3. Use bit flags for boolean state (pack into uint32)
4. Profile and tune

## Memory Analysis

### Current Context

```go
type LineBreakContext struct {
    text   string           // 16 bytes (pointer + length)
    runes  []rune           // 24 bytes (pointer + len + cap)
    classes []BreakClass    // 24 bytes (now uint8 after Phase 1)
    hyphens Hyphens         // 1 byte
    pos int                 // 8 bytes
    prevClass BreakClass    // 1 byte (after Phase 1)
    currClass BreakClass    // 1 byte
    nextClass BreakClass    // 1 byte
    lastNonSpaceClass BreakClass  // 1 byte
    bytePositions []int     // 24 bytes
}
// Total: ~101 bytes + slice backing arrays
```

### With Environment

```go
type LineBreakEnvironment struct {
    quoteStack [8]QuoteContext  // 8 * 16 bytes = 128 bytes (pre-allocated)
    quoteTop uint8              // 1 byte (stack pointer)
    riCount uint8               // 1 byte
    riStartPos int16            // 2 bytes (int16 sufficient for position delta)
    lastHLPos int16             // 2 bytes
    parenDepth uint8            // 1 byte
    bracketDepth uint8          // 1 byte
    inAksaraSequence bool       // 1 byte
    aksaraStartPos int16        // 2 bytes
    lastNonSpacePos int16       // 2 bytes
    lastNonSpaceClass BreakClass // 1 byte
}
// Total: ~142 bytes (all stack-allocated, zero heap)

type LineBreakContext struct {
    // ... existing fields ...
    env *LineBreakEnvironment   // 8 bytes pointer
}
// Total: ~109 bytes + ~142 bytes environment = ~251 bytes
```

**Trade-off**: +150 bytes per context, but **zero backward scanning** (massive CPU savings).

## Performance Projection

### Current Performance (After Phase 1 + Phase 2)

- Short (10 chars): 1,256 ns/op (~2.3x slower)
- Medium (64 chars): 9,021 ns/op (~2.3x slower)
- Long (45 chars): 4,963 ns/op (~2.3x slower)

### Expected After Streaming Parser

**Conservative estimate**: 1.8-2.0x improvement
- Eliminates backward scanning (O(N²) → O(N))
- Reduces average rules checked per character (~10 → ~6)
- Better branch prediction (forward-only)

**Best case**: 2.5-3.0x improvement if environment lookups are highly cache-friendly

### Target

- Short (10 chars): **600-700 ns/op** (1.0-1.2x slower than original)
- Medium (64 chars): **4,500-5,000 ns/op** (1.0-1.2x slower)
- Long (45 chars): **2,500-2,800 ns/op** (1.0-1.2x slower)

## Testing Strategy

1. **Add environment structure** → Test: 100% conformance maintained
2. **Port quote rules** → Test: 100% conformance maintained
3. **Port RI rules** → Test: 100% conformance maintained
4. **Port Hebrew rules** → Test: 100% conformance maintained
5. **Port all remaining rules** → Test: 100% conformance maintained
6. **Optimize environment updates** → Benchmark: Measure improvement
7. **Final verification** → Test: 19,338/19,338 passing

## Risks and Mitigations

### Risk 1: Environment State Bugs

**Problem**: Forgetting to update environment can cause subtle bugs

**Mitigation**:
- Comprehensive test suite (already have 19,338 tests)
- Add debug mode that validates environment state
- Test after each rule category port

### Risk 2: Increased Memory Footprint

**Problem**: +150 bytes per context might be significant for embedded systems

**Mitigation**:
- Make streaming parser opt-in: `FindLineBreakOpportunitiesStreaming()`
- Keep original implementation for memory-constrained environments
- Document trade-offs clearly

### Risk 3: Complexity Increase

**Problem**: More state to track means more complex debugging

**Mitigation**:
- Add `DebugEnv()` method that prints environment state
- Document environment invariants clearly
- Use clear naming for all environment fields

## Success Criteria

1. ✅ **100% conformance maintained** (19,338/19,338 tests)
2. ✅ **1.8x+ performance improvement** (streaming vs current)
3. ✅ **No heap allocations** in hot path (environment is stack-allocated)
4. ✅ **Zero backward scanning** (all rules use forward-only lookups)
5. ✅ **Code clarity** (environment makes rule logic clearer, not more obscure)

If any criteria fails, we keep both implementations and document trade-offs.

## Implementation Timeline

**Estimated effort**: 4-6 hours of focused work

1. **Hour 1**: Add LineBreakEnvironment struct and integration (Phase 1)
2. **Hour 2**: Port quote rules (LB19, LB15)
3. **Hour 3**: Port RI rules (LB30a)
4. **Hour 4**: Port Hebrew rules (LB21a), space rules (LB18)
5. **Hour 5**: Test comprehensive conformance, fix bugs
6. **Hour 6**: Optimize and benchmark

## Next Steps

1. Implement `LineBreakEnvironment` struct in context.go
2. Add environment field to `LineBreakContext`
3. Initialize in `NewLineBreakContext()`
4. Add environment update logic to `Slide()`
5. Test: Verify 100% conformance maintained with no-op environment
6. Begin porting rules to use environment

Ready to begin implementation?
