# UAX #14 Performance Optimization Plan

## Current Performance Bottlenecks

### 1. Virtual Function Call Overhead ⚠️ **HIGH IMPACT**

**Problem**: Every character position loops through 59 rule functions via function pointers:

```go
for _, rule := range lineBreakRules {
    if matched, ruleDecision := rule(ctx); matched {
        decision = ruleDecision
        break
    }
}
```

**Impact**: Function pointer indirection is slower than direct calls
- Average case: ~10-15 rule calls per character (until match)
- Worst case: All 59 rules checked

**Solution Options**:

#### Option A: Fast-Path Switch Statement
```go
// Handle 90% of cases with direct calls (no function pointers)
switch {
case prev == ClassSP:
    decision = handleSpaceBreaks(ctx, prev, curr)
case curr == ClassSP:
    decision = BreakNo
case prev == ClassAL && curr == ClassAL:
    decision = BreakNo
// ... top 10-15 patterns
default:
    // Fall back to rule loop for complex cases
    for _, rule := range lineBreakRules {
        if matched, ruleDecision := rule(ctx); matched {
            decision = ruleDecision
            break
        }
    }
}
```

**Estimated gain**: 1.5-2x faster (reduces 2.5x to 1.2-1.5x slower)

#### Option B: Rule Priority Buckets
```go
// Group rules by frequency
var fastRules = []LineBreakRule{ruleLB18, ruleLB28, ruleLB29, ...}  // 80% hit rate
var normalRules = []LineBreakRule{ruleLB14, ruleLB15, ...}         // 15% hit rate
var rareRules = []LineBreakRule{ruleLB19_German, ...}              // 5% hit rate

// Check fast rules first
for _, rule := range fastRules {
    if matched, decision := rule(ctx); matched {
        return decision
    }
}
// Then normal rules, then rare rules
```

**Estimated gain**: 1.3-1.5x faster

### 2. No Caching 🔄 **MEDIUM IMPACT**

**Problem**: Same character pairs checked repeatedly

```go
// "hello world" - 'l' × 'l' checked 2 times, ' ' × 'w' same as ' ' × 'h'
```

**Solution**: LRU cache for (prevClass, currClass) → decision

```go
type BreakCache struct {
    entries [256]CacheEntry  // Pre-allocated, no heap
}

type CacheEntry struct {
    prev     BreakClass
    curr     BreakClass
    decision BreakDecision
    valid    bool
}

func (c *BreakCache) Get(prev, curr BreakClass) (BreakDecision, bool) {
    idx := (int(prev) ^ int(curr)) & 0xFF
    entry := c.entries[idx]
    if entry.valid && entry.prev == prev && entry.curr == curr {
        return entry.decision, true
    }
    return BreakNo, false
}
```

**Estimated gain**: 1.2-1.3x faster for repetitive text

### 3. Repeated Context Method Calls 📞 **LOW-MEDIUM IMPACT**

**Problem**: Hot-path methods called many times per character

**Status**: ✅ **FIXED** - Added `//go:inline` directives to:
- `Pos()`, `Prev()`, `Curr()`, `Next()`
- `Rune()`, `RuneAt()`, `ClassAt()`
- `LastNonSpace()`, `Len()`, `Hyphens()`

**Estimated gain**: 1.1-1.2x faster (compiler will inline these)

### 4. Backward Scanning 🔍 **HIGH IMPACT (future)**

**Problem**: Rules like `ruleLB19_German` scan backward to find opening quotes

```go
for checkIdx := pos - 3; checkIdx >= 0 && checkIdx > pos-30; checkIdx-- {
    // Scan backward
}
```

**Solution**: Streaming parser with environment (see proposal above)
- Track quote stack, bracket stack, hyphen state in forward pass
- Zero backward scanning

**Estimated gain**: 2-3x faster (brings us to original speed or faster)
- Requires architectural change (v5.1.0 or v6.0.0)

## Optimization Priority

### Phase 1: Quick Wins (v5.0.1) - **Achieved: 1.04-1.07x improvement**
1. ✅ Add `//go:inline` directives (DONE)
2. ✅ Bitpack enums (int → uint8) (DONE - 4-7% faster, 8x memory reduction)
3. ❌ Fast-path switch (ABANDONED - broke conformance)

### Phase 2: Caching (v5.0.2) - **Target: Additional 1.2x**
1. Implement simple break decision cache
2. Profile cache hit rates
3. Tune cache size

### Phase 3: Streaming Parser (v5.1.0 or v6.0.0) - **Target: 2-3x**
1. Design `LineBreakEnvironment`
2. Implement state tracking
3. Port rules to use environment
4. Maintain 100% conformance

## Combined Potential

Starting: **2.5x slower** than original

After all optimizations: **0.8-1.0x** (potentially faster than original!)

- Phase 1: ✅ 2.5x → 2.4x slower (bitpacking: 1.05x improvement)
- Phase 2: ⏳ 2.4x → 1.7x slower (flat array: expected 1.4x improvement)
- Phase 3: ⏳ 1.7x → 1.4x slower (profile-guided: expected 1.2x improvement)
- Phase 4: ⏳ 1.4x → 1.0x slower (streaming parser: expected 1.4x improvement)

## Benchmarking Commands

```bash
# Current performance
go test -bench=BenchmarkRulesVsOriginal -benchmem -benchtime=3s

# Profile hot spots
go test -bench=BenchmarkRulesVsOriginal -cpuprofile=cpu.prof -benchtime=3s
go tool pprof -http=:8080 cpu.prof

# Inline analysis
go build -gcflags='-m' 2>&1 | grep context.go

# Assembly inspection
go tool compile -S context.go | grep -A5 "Prev"
```

## Implementation Notes

- Maintain 100% conformance (19,338/19,338 tests)
- Keep rule-based architecture for maintainability
- Optimizations should be transparent to users
- Benchmark before and after each change
- Document performance trade-offs in commit messages
