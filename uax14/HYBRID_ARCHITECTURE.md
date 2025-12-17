# Hybrid Architecture Design: Pair Table First

## Problem Statement

Current architecture checks 44 rules before falling back to pair table:
- **83.76% of cases** hit the pair table (checked LAST)
- **16.24% are rule exceptions** (checked FIRST via 44 rules)
- Average **38.5 rule checks** per position before pair table

This is backwards! We're paying the cost of checking ~38 rules even though 84% of cases don't need them.

## Proposed Solution

**Invert the dispatch order**: Check pair table FIRST, only run rules when necessary.

### Step 1: Add New BreakAction

```go
const (
    BreakProhibited BreakAction = iota  // Don't break (×)
    BreakDirect                          // Break allowed (÷)
    BreakIndirect                        // Break if SP present (÷ after SP)
    BreakMandatory                       // Must break (!)
    BreakCheckRules                      // Need to check rule exceptions (NEW!)
)
```

### Step 2: Mark Exception Pairs

After populating the pair table, mark (prev, curr) pairs that have rule exceptions:

```go
func init() {
    // Populate pair table from map...

    // Mark pairs that need rule checking
    markRuleExceptionPairs()
}

func markRuleExceptionPairs() {
    // Analyze each rule to determine which (prev, curr) pairs it handles
    // Mark those pairs with BreakCheckRules

    // Example: LB12c handles (BA, GL)
    pairTableFlat[ClassBA][ClassGL] = BreakCheckRules

    // Example: LB13 handles (*, CL|CP|EX|IS|SY)
    for prev := BreakClass(0); prev < 65; prev++ {
        pairTableFlat[prev][ClassCL] = BreakCheckRules
        pairTableFlat[prev][ClassCP] = BreakCheckRules
        pairTableFlat[prev][ClassEX] = BreakCheckRules
        pairTableFlat[prev][ClassIS] = BreakCheckRules
        pairTableFlat[prev][ClassSY] = BreakCheckRules
    }

    // ... mark all other rule exceptions
}
```

### Step 3: Invert Dispatch Logic

```go
// Current (slow):
// 1. Check 7 inlined rules
// 2. Check 37 function pointer rules
// 3. Fall back to pair table (84% of cases!)

// Proposed (fast):
// 1. Check pair table (84% exit here immediately!)
// 2. If BreakCheckRules, check 7 inlined rules
// 3. If still no match, check 37 function pointer rules
```

## Implementation Strategy

### Phase 1: Identify Rule Exception Pairs

Analyze each rule to determine which (prev, curr) class pairs it needs:

#### Rules with Specific Prev Class
- LB4: `(ClassBK, *)`
- LB5a: `(ClassCR, ClassLF)`
- LB5b: `(ClassCR|LF|NL, *)`
- LB7: `(*, ClassZW)`
- LB11: `(ClassWJ, *)` OR `(*, ClassWJ)`
- LB12: `(ClassGL, *)`

#### Rules with Pattern Matching
- LB13: `(*, ClassCL|CP|EX|IS|SY)` - 5 curr classes × 65 prev classes = 325 pairs
- LB14: `(ClassOP [+SP*], *)` - needs backward scan, mark `(ClassOP, *)` and `(ClassSP, *)` when prev context has OP
- LB25: `(ClassNU, *)` and complex numeric patterns

#### Rules with Backward Scanning
- LB8: ZW SP* ÷ - scans backward for ZW
- LB14: OP SP* × - scans backward for OP
- LB19 variants: Complex quotation patterns with backward scans

**Challenge**: Rules with backward scanning can't be captured by simple (prev, curr) pairs!

### Phase 2: Conservative Marking Strategy

**Problem**: Many rules scan backward past SP, CM, ZWJ, etc. They depend on context beyond just (prev, curr).

**Solution 1 (Conservative)**: Mark ALL pairs involving classes that appear in any rule
- Safe but defeats the optimization
- Would mark most of the 65×65 = 4,225 possible pairs

**Solution 2 (Aggressive)**: Only mark pairs where rules definitely apply
- Risk: Miss some cases and break conformance
- Phase 7c showed this breaks 399 tests (2.1%)

**Solution 3 (Hybrid - Recommended)**:
- Fast path: Check pair table for common cases (alphabetics, etc.)
- Always check rules for "complex" classes:
  - Quotation marks: QU, QU_Pi, QU_Pf
  - Spaces: SP (rules scan backward past SP)
  - Combining marks: CM, ZWJ
  - Numeric: NU, SY, IS
  - Punctuation: CL, CP, OP, EX
  - Hebrew: HL, HH

## Estimated Impact

### Best Case (Solution 2 - Aggressive)
If we could accurately mark only the 16.24% of pairs that need rules:
- 83.76% of cases: 1 pair table lookup → **instant decision**
- 16.24% of cases: 1 pair table lookup + rule checking → same as before

**Expected improvement: 3-5x faster** (eliminates 38.5 rule checks in 84% of cases)

### Realistic Case (Solution 3 - Hybrid)
Mark 30-40% of pairs as needing rule checks (conservative):
- 60-70% of cases: Pair table only → **2-3x faster**
- 30-40% of cases: Pair table + rules → same as before

**Expected improvement: 1.5-2x faster**

### Worst Case (Solution 1 - Conservative)
Mark 80%+ of pairs as needing checks:
- Minimal benefit, adds pair table lookup overhead
- **Expected: 5-10% slower than current**

## Implementation Plan

1. **Add BreakCheckRules constant**
2. **Profile existing tests**: Record which (prev, curr) pairs hit rules vs pair table
3. **Mark conservative exception set**: Classes that appear in rules
4. **Test conformance**: All 19,338 tests must pass
5. **Benchmark**: Measure improvement
6. **Iterate**: If safe, try marking fewer pairs

## Risk Assessment

**High Risk**: Missing edge cases with backward scanning
- LB8: ZW SP* ÷ - needs to detect ZW 2+ positions back
- LB14: OP SP* × - needs to detect OP 2+ positions back
- LB19: Quote matching - complex backward scans

**Mitigation**: Always mark (*, SP), (*, CM), (*, ZWJ) as needing rule checks, since many rules scan past these.

## Alternative: Runtime Profiling

Instead of static marking, use runtime profiling:
1. Start with all pairs marked as BreakCheckRules
2. Record which pairs actually match rules vs fall through to pair table
3. After warmup period, unmark pairs that never match rules
4. Dynamically optimize based on actual text patterns

**Pros**: Adapts to actual usage patterns
**Cons**: Complex, requires warmup, thread-safety concerns
