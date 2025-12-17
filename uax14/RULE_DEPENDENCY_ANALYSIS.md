# UAX #14 Rule Dependency Analysis

## Profiling Results Summary

**Test corpus**: 19,338 conformance tests (41,149 positions)
**Average rule checks per position**: 38.5
**Pair table hit rate**: 83.76% (but checked LAST)

### Top Rules by Match Frequency

| Rank | Rule | Matches | % Total | Hit Rate | Current Index |
|------|------|---------|---------|----------|---------------|
| 1 | PairTable | 34,467 | 83.76% | 100.00% | (fallback) |
| 2 | **LB13** | 2,157 | 5.24% | 5.72% | **11** |
| 3 | **LB5b** | 833 | 2.02% | 2.04% | **2** |
| 4 | **LB14** | 679 | 1.65% | 1.91% | **15** |
| 5 | **LB12** | 563 | 1.37% | 1.46% | **7** |
| 6 | **LB11** | 546 | 1.33% | 1.39% | **6** |
| 7 | LB4 | 282 | 0.69% | 0.69% | 0 |
| 8 | LB8a | 280 | 0.68% | 0.70% | 4 |
| 9 | LB7 | 276 | 0.67% | 0.69% | 3 |
| 10 | LB8 | 275 | 0.67% | 0.70% | 5 |
| 11 | LB12a | 260 | 0.63% | 0.68% | 9 |
| 12 | LB19_SP_QU_Pf | 153 | 0.37% | 0.44% | 22 |
| 13 | LB25 | 95 | 0.23% | 0.25% | 10 |

### Observations

1. **LB13 is the highest-hit rule** (5.24%) but sits at index 11
2. **LB5b is already well-positioned** at index 2 (2.02% hit rate)
3. **LB14 is at index 15** despite being #4 in hit rate (1.65%)
4. **83.76% of decisions come from pair table** checked at the end

## Current Rule Ordering

```go
var lineBreakRules = []LineBreakRule{
    // [0-2] Mandatory breaks (MUST be first per UAX #14)
    ruleLB4,                // 0: BK ÷ (0.69%)
    ruleLB5a,               // 1: CR × LF (0.005% - rare!)
    ruleLB5b,               // 2: CR ÷, LF ÷, NL ÷ (2.02% - #3!)

    // [3-5] Zero-width characters (MUST come before general rules)
    ruleLB7,                // 3: × ZW (0.67%)
    ruleLB8a,               // 4: ZWJ × (0.68%)
    ruleLB8,                // 5: ZW SP* ÷ (0.67%)

    // [6-9] Word joiner and glue
    ruleLB11,               // 6: WJ ×, × WJ (1.33% - #6!)
    ruleLB12,               // 7: GL × (1.37% - #5!)
    ruleLB12c,              // 8: BA ÷ GL (0.019% - rare!)
    ruleLB12a,              // 9: [^SP BA HY] × GL (0.63%)

    // [10] Numeric expressions - MUST come before LB13
    ruleLB25,               // 10: NU (SY | IS)* × NU (0.23%)

    // [11] Closing punctuation
    ruleLB13,               // 11: × [CL CP EX IS SY] (5.24% - HIGHEST!)

    // [12-14] Quotation marks - MUST come before LB14/LB15
    ruleLB19_Guillemet,     // 12: (0.002% - rare!)
    ruleLB19_German,        // 13: (0.005% - rare!)
    ruleLB19_QU_Pi_SP,      // 14: (not in top 20)

    // [15-16] Opening punctuation
    ruleLB14,               // 15: OP SP* × (1.65% - #4!)
    ruleLB15,               // 16: QU SP* × OP (0.007% - rare!)

    // [17-18] More closing/B2
    ruleLB16,               // 17: (CL | CP) SP* × NS (0.032%)
    ruleLB17,               // 18: B2 SP* × B2 (0.005% - rare!)

    // [19-22] More quotation patterns
    ruleLB19_NS_QU_Pi,      // 19: (0.015%)
    ruleLB19_CJK_QU_Pf_ID,  // 20: (0.012%)
    ruleLB19_CJK_ID_QU_Pi,  // 21: (0.007%)
    ruleLB19_SP_QU_Pf,      // 22: (0.37% - #12!)

    // [23-43] Lower frequency rules...
    ruleLB20,               // 23: ÷ CB, CB ÷ (0.11%)
    ruleLB21_HY,            // 24: (0.022%)
    // ... (all < 0.1%)
}
```

## Identified Constraints (UAX #14 Spec)

### Hard Constraints (Cannot Violate)

1. **LB4-LB5 MUST be first** (mandatory breaks)
   - These override ALL other rules
   - Spec section 6.1: "LB4 and LB5 are first"

2. **LB7, LB8, LB8a MUST come early** (zero-width)
   - These affect character properties before other rules apply
   - LB8 MUST come before LB11 (proven in Phase 7a)

3. **LB25 MUST come before LB13**
   - LB25 handles numeric expressions with leading decimals
   - LB13 would incorrectly break at decimal points
   - Comment in code: "must come before LB13"

4. **LB19 variants MUST come before LB14/LB15**
   - LB19 handles exception cases for quotation marks
   - LB14/LB15 are general opening punctuation rules
   - Comment in code: "MUST come before LB14/LB15 to handle exceptions"

5. **LB12c MUST come before LB12a**
   - LB12c is exception (BA ÷ GL)
   - LB12a is general rule ([^SP BA HY] × GL)
   - Exception must be checked first

### Soft Constraints (Ordering Within Groups)

1. **Within [0-2] (Mandatory breaks)**
   - Current: LB4 (0.69%), LB5a (0.005%), LB5b (2.02%)
   - Optimal: LB5b, LB4, LB5a
   - **Can reorder!** These are independent checks

2. **Within [3-5] (Zero-width)**
   - Current: LB7 (0.67%), LB8a (0.68%), LB8 (0.67%)
   - All similar hit rates (~0.67-0.70%)
   - Constraint: LB8 must come before LB11 (in next group)
   - **Minimal benefit to reorder**

3. **Within [6-9] (Word joiner/glue)**
   - Current: LB11 (1.33%), LB12 (1.37%), LB12c (0.019%), LB12a (0.63%)
   - Optimal: LB12, LB11, LB12a, LB12c
   - Constraint: LB12c MUST come before LB12a
   - **Can partially reorder!** LB12 and LB11 can swap

4. **Within [12-14] (LB19 quotation variants)**
   - All have very low hit rates (< 0.01%)
   - LB19_SP_QU_Pf (at index 22) has 0.37% - much higher!
   - **Might be able to move LB19_SP_QU_Pf earlier** if independent

5. **Within [15-18] (Opening/closing punct)**
   - LB14 (1.65%) vs LB15 (0.007%)
   - Already in optimal order
   - **No change needed**

## Reordering Opportunities

### Opportunity 1: Reorder Mandatory Breaks [0-2]

**Current:**
```go
ruleLB4,   // 0: BK ÷ (0.69%)
ruleLB5a,  // 1: CR × LF (0.005%)
ruleLB5b,  // 2: CR ÷, LF ÷, NL ÷ (2.02%)
```

**Proposed:**
```go
ruleLB5b,  // 0: CR ÷, LF ÷, NL ÷ (2.02% - move to first!)
ruleLB4,   // 1: BK ÷ (0.69%)
ruleLB5a,  // 2: CR × LF (0.005%)
```

**Rationale:**
- LB5b has 2.02% hit rate (3x higher than LB4)
- These rules are independent (check different classes)
- LB5a can go last (only 2 matches in 41,149 positions!)

**Safety:** LB5a (CR × LF) should come BEFORE LB5b to prevent incorrect breaks in CR LF sequences.

**Corrected proposal:**
```go
ruleLB5a,  // 0: CR × LF (0.005% but PREVENTS LB5b from breaking CR LF)
ruleLB5b,  // 1: CR ÷, LF ÷, NL ÷ (2.02%)
ruleLB4,   // 2: BK ÷ (0.69%)
```

**Wait - this is wrong!** Let me reconsider...

Actually, looking at the logic:
- LB4: `if prevClass == ClassBK { return true, BreakYes }`
- LB5a: `if prevClass == ClassCR && currClass == ClassLF { return true, BreakNo }`
- LB5b: `if prevClass in {CR, LF, NL} { return true, BreakYes }`

LB5a MUST come before LB5b to prevent LB5b from breaking CR LF pairs!

**Final corrected order:**
```go
ruleLB5a,  // 0: CR × LF (0.005% - low hit but MUST be first)
ruleLB5b,  // 1: CR ÷, LF ÷, NL ÷ (2.02% - high hit!)
ruleLB4,   // 2: BK ÷ (0.69%)
```

### Opportunity 2: Swap LB12 and LB11 [6-7]

**Current:**
```go
ruleLB11,  // 6: WJ ×, × WJ (1.33%)
ruleLB12,  // 7: GL × (1.37%)
```

**Proposed:**
```go
ruleLB12,  // 6: GL × (1.37%)
ruleLB11,  // 7: WJ ×, × WJ (1.33%)
```

**Rationale:**
- LB12 has slightly higher hit rate (1.37% vs 1.33%)
- These rules check different classes (GL vs WJ)
- LB8 constraint is satisfied (it comes before both)

**Safety:** Need to verify these are independent in the spec.

### Opportunity 3: LB12c vs LB12a

**Current:**
```go
ruleLB12c,  // 8: BA ÷ GL (0.019% - rare!)
ruleLB12a,  // 9: [^SP BA HY] × GL (0.63%)
```

**Proposed:** Keep as-is (exception before general rule)

**Rationale:** LB12c is exception to LB12a - MUST come first

### Opportunity 4: Cannot Move LB13 Earlier

**Problem:** LB13 (5.24% - highest!) sits at index 11

**Constraint:** LB25 (0.23%) must come before LB13

**Analysis:** LB25 handles patterns like ".5" (leading decimal) which need special treatment before LB13's general closing punctuation rule.

**Conclusion:** **CANNOT move LB13 earlier** without violating LB25 dependency

### Opportunity 5: Cannot Move LB14 Earlier

**Problem:** LB14 (1.65% - #4) sits at index 15

**Constraint:** LB19 variants (12-14) must come before LB14

**Analysis:** LB19 handles quotation mark exceptions that override LB14's general opening punctuation rule.

**Conclusion:** **CANNOT move LB14 earlier** without violating LB19 dependencies

## Estimated Impact

### Scenario: Reorder opportunities 1 + 2

**Changes:**
1. Swap LB5a/LB5b/LB4 → LB5a, LB5b, LB4
2. Swap LB11/LB12 → LB12, LB11

**Expected savings:**
- LB5b moves from index 2 → 1: Saves 1 check for 2.02% of positions = **0.02 checks/position**
- LB12 moves from index 7 → 6: Saves 1 check for 1.37% of positions = **0.014 checks/position**
- Total: **0.034 checks/position saved**

**Current:** 38.5 checks/position
**After:** 38.47 checks/position
**Improvement:** **0.09% reduction in rule checks** (negligible!)

## Fundamental Limitation

The real problem is **architectural**, not ordering:

1. **83.76% of cases hit the pair table** (checked last)
2. **16.24% are exceptions** (checked first via rules)
3. Average 38.5 rule checks means we check **most/all 44 rules** before pair table

The only way to significantly improve would be:
1. **Check pair table first** - but Phase 7c proved this breaks 399 tests (rules are exceptions!)
2. **Rule bucketing** - group rules by character class constraints
3. **Decision tree** - build a decision tree based on (prev, curr) classes

## Recommendation

**Micro-optimization:** Implement opportunities 1 and 2 for code cleanliness (put higher-hit rules first within independent groups), but expect **< 0.1% performance improvement**.

**Macro-optimization:** Focus on other areas:
1. **Expand ASCII fast path** to cover more punctuation (30-40x speedup potential)
2. **Class lookup caching** (every character does class lookup)
3. **Real-world corpus profiling** (Wikipedia/GitHub data may show different distributions)
4. **Rule bucketing by class constraints** (check only rules relevant to current (prev, curr) pair)
