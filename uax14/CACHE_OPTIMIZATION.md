# Cache-Friendly Optimization Plan

## Current Memory Layout Analysis

### Data Structures

```go
type BreakClass int        // 8 bytes (64-bit int)
type BreakAction int       // 8 bytes (64-bit int)

var pairTable = map[[2]BreakClass]BreakAction{...}
// Map with 16-byte keys ([2]int64) and 8-byte values
// ~2,064 entries × 24 bytes (key+value) = ~50KB
// Plus map overhead (buckets, hashing, pointers)
```

### Performance Issues

1. **Excessive memory per enum**: 8 bytes for ~50 BreakClass values (needs only 6 bits)
2. **Map lookup overhead**: Hash calculation + bucket traversal + pointer chasing
3. **Cache unfriendly**: Map entries scattered in memory, poor spatial locality
4. **Large memory footprint**: ~50KB+ for something that could fit in 2.5KB

## Proposed Optimizations

### Phase 1: Bitpack Enums (Safe, Zero Risk)

```go
type BreakClass uint8      // 1 byte (can represent 0-255)
type BreakAction uint8     // 1 byte (only need 0-3)

const (
    BreakProhibited BreakAction = 0  // 0b00
    BreakDirect     BreakAction = 1  // 0b01
    BreakIndirect   BreakAction = 2  // 0b10
    BreakMandatory  BreakAction = 3  // 0b11
)
```

**Memory savings**:
- Context.prevClass/currClass/nextClass: 24 bytes → 3 bytes (8x smaller)
- Context.classes slice: N×8 bytes → N×1 bytes (8x smaller for N runes)
- Pair table keys: 16 bytes → 2 bytes (8x smaller)

**Expected gain**: 1.2-1.4x faster (better cache utilization)

### Phase 2: Flat Array Pair Table (Medium Risk)

Replace hash map with direct-indexed array:

```go
// Flat 2D array: 50×50 = 2,500 entries × 1 byte = 2.5KB (fits in L1 cache!)
var pairTableFlat [256][256]BreakAction  // Pre-sized to max uint8

func getBreakAction(before, after BreakClass) BreakAction {
    // Direct array lookup - no hashing, no collisions
    action := pairTableFlat[before][after]
    if action != 0 {
        return action
    }

    // Try wildcard patterns
    action = pairTableFlat[before][ClassXX]
    if action != 0 {
        return action
    }
    action = pairTableFlat[ClassXX][after]
    if action != 0 {
        return action
    }

    // Default rules (inline for speed)
    if before == ClassSP {
        return BreakIndirect
    }
    if after == ClassSP {
        return BreakProhibited
    }
    return BreakDirect
}
```

**Memory**: 256×256 = 65KB (still fits in L2 cache)

**Expected gain**: 1.3-1.5x faster (eliminates map overhead)

**Risk**: Must verify all 19,338 tests still pass

### Phase 3: Profile-Guided Optimization (Low Risk)

Profile real text to identify hot character pairs, then add fast-path:

```go
// Profile shows these pairs are 80% of cases
func getBreakActionFast(before, after BreakClass) BreakAction {
    // Inline most common cases (no function call overhead)
    if before == ClassAL && after == ClassAL { return BreakProhibited }
    if before == ClassAL && after == ClassSP { return BreakProhibited }
    if before == ClassSP && after == ClassAL { return BreakIndirect }
    if before == ClassNU && after == ClassNU { return BreakProhibited }
    // ... top 10-15 pairs

    // Fall back to array lookup
    return getBreakAction(before, after)
}
```

**Expected gain**: 1.1-1.2x faster (reduces array lookups for common text)

### Phase 4: Extreme Bitpacking (High Risk, Experimental)

Pack 4 BreakAction values per byte (2 bits each):

```go
var pairTablePacked [256][64]byte  // 256×64 = 16KB (L1 cache friendly!)

func getBreakAction(before, after BreakClass) BreakAction {
    byteIdx := after >> 2       // Divide by 4
    bitOffset := (after & 0x3) << 1  // Modulo 4, multiply by 2
    packedByte := pairTablePacked[before][byteIdx]
    return BreakAction((packedByte >> bitOffset) & 0x3)
}
```

**Memory**: 16KB (extremely cache friendly)

**Expected gain**: 1.1-1.2x faster (better cache utilization)

**Risk**: More complex code, harder to debug

## Combined Performance Projection

Starting: **2.5x slower** than original

| Phase | Optimization | Target | Cumulative |
|-------|-------------|--------|------------|
| 1 | Bitpack enums | 1.3x faster | **1.9x slower** |
| 2 | Flat array table | 1.4x faster | **1.4x slower** |
| 3 | Profile-guided fast-path | 1.2x faster | **1.2x slower** |
| 4 | Extreme bitpacking (optional) | 1.2x faster | **1.0x slower** |

With streaming parser (future): **0.8-1.0x** (potentially faster than original!)

## Implementation Plan

### Phase 1: Bitpack Enums (v5.0.1)
1. Change `type BreakClass int` → `type BreakClass uint8`
2. Change `type BreakAction int` → `type BreakAction uint8`
3. Run all tests to verify 100% conformance maintained
4. Benchmark to measure improvement
5. Commit if successful

**Estimated time**: 30 minutes
**Risk level**: Very low (just type change)

### Phase 2: Flat Array (v5.0.2)
1. Pre-compute flat array from current map
2. Replace `map[[2]BreakClass]BreakAction` with `[256][256]BreakAction`
3. Update `getBreakAction()` to use array indexing
4. Run all tests (critical - must maintain 100%)
5. Benchmark to verify improvement
6. Commit if successful

**Estimated time**: 1 hour
**Risk level**: Medium (must verify correctness)

### Phase 3: Profile-Guided Optimization (v5.0.3)
1. Run `/tmp/profile_pairs.go` on diverse text samples
2. Identify top 10-20 character pair patterns
3. Add inline fast-path checks for hot pairs
4. Benchmark and verify improvement
5. Commit if successful

**Estimated time**: 1 hour
**Risk level**: Low (additive optimization)

### Phase 4: Extreme Bitpacking (v5.1.0 or skip)
1. Implement bitpacked pair table
2. Extensive testing and benchmarking
3. Compare complexity vs performance gain
4. Decide if worth the maintenance cost

**Estimated time**: 2-3 hours
**Risk level**: High (complex bit manipulation)

## Benchmarking Commands

```bash
# Baseline
git checkout v5.0.0
go test -bench=BenchmarkRulesVsOriginal -benchmem -benchtime=3s > baseline.txt

# After each phase
go test -bench=BenchmarkRulesVsOriginal -benchmem -benchtime=3s > phase1.txt
benchstat baseline.txt phase1.txt

# Profile memory allocations
go test -bench=BenchmarkRulesVsOriginal -memprofile=mem.prof
go tool pprof -http=:8080 mem.prof

# Profile CPU hotspots
go test -bench=BenchmarkRulesVsOriginal -cpuprofile=cpu.prof
go tool pprof -http=:8080 cpu.prof
```

## Success Criteria

Each phase must satisfy:
1. ✅ 100% conformance (19,338/19,338 tests passing)
2. ✅ Measurable performance improvement (>10% faster)
3. ✅ No increased memory allocations
4. ✅ Code remains maintainable and documented

If any phase fails these criteria, revert and document findings.
