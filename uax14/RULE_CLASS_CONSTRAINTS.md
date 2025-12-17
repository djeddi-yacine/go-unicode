# UAX #14 Rule Class Constraints Analysis

## Purpose

Analyze each rule to identify which BreakClass values they check for `prev` and `curr` positions.
This enables rule bucketing: only check rules relevant to the current (prev, curr) class pair.

## Methodology

For each rule, extract:
1. **Prev constraints**: Which `prev` classes trigger this rule
2. **Curr constraints**: Which `curr` classes trigger this rule
3. **Always check**: Rules with no class constraints or complex lookback logic

## Rule Analysis

### Group 1: Mandatory Breaks (Always Check - High Priority)

#### LB4: BK ÷
- **Prev**: `ClassBK`
- **Curr**: Any
- **Logic**: `if prev == ClassBK { return true, BreakYes }`
- **Bucket**: `prev == ClassBK`

#### LB5a: CR × LF
- **Prev**: `ClassCR`
- **Curr**: `ClassLF`
- **Logic**: `if prev == ClassCR && curr == ClassLF { return true, BreakNo }`
- **Bucket**: `prev == ClassCR && curr == ClassLF`

#### LB5b: CR ÷, LF ÷, NL ÷
- **Prev**: `ClassCR | ClassLF | ClassNL`
- **Curr**: Any
- **Logic**: `if prev in {CR, LF, NL} { return true, BreakYes }`
- **Bucket**: `prev in {ClassCR, ClassLF, ClassNL}`

### Group 2: Zero-Width Characters (Always Check - High Priority)

#### LB7: × ZW
- **Prev**: Any
- **Curr**: `ClassZW`
- **Logic**: `if curr == ClassZW { return true, BreakNo }`
- **Bucket**: `curr == ClassZW`

#### LB8a: ZWJ ×
- **Prev**: ZWJ rune (U+200D) - not class!
- **Curr**: Any
- **Logic**: `if i > 0 && runes[i-1] == '\u200D' { return true, BreakNo }`
- **Bucket**: **Always check** (checks rune, not class)

#### LB8: ZW SP* ÷
- **Prev**: Complex (scans backward for ZW past SP)
- **Curr**: Any
- **Logic**: Scans backward from prev position looking for ZW
- **Bucket**: **Always check** (has backward scan)

### Group 3: Word Joiner and Glue

#### LB11: WJ ×, × WJ
- **Prev**: `ClassWJ` OR
- **Curr**: `ClassWJ`
- **Logic**: `if prev == ClassWJ || curr == ClassWJ { return true, BreakNo }`
- **Bucket**: `prev == ClassWJ || curr == ClassWJ`

#### LB12: GL ×
- **Prev**: `ClassGL`
- **Curr**: Any
- **Logic**: `if prev == ClassGL { return true, BreakNo }`
- **Bucket**: `prev == ClassGL`

#### LB12c: BA ÷ GL (exception)
- **Prev**: `ClassBA`
- **Curr**: `ClassGL`
- **Logic**: `if prev == ClassBA && curr == ClassGL { return true, BreakYes }`
- **Bucket**: `prev == ClassBA && curr == ClassGL`

#### LB12a: [^SP BA HY] × GL
- **Prev**: NOT `ClassSP | ClassBA | ClassHY`
- **Curr**: `ClassGL`
- **Logic**: `if curr == ClassGL && prev not in {SP, BA, HY} { return true, BreakNo }`
- **Bucket**: `curr == ClassGL` (need to exclude BA already handled)

### Group 4: Numeric Expressions

#### LB25: NU (SY | IS)* × NU, etc.
- **Prev**: Complex patterns involving `ClassNU, ClassCL, ClassCP, ClassSY, ClassIS`
- **Curr**: `ClassNU` in most cases
- **Logic**: Multiple patterns with backward scanning
- **Bucket**: **Always check** OR bucket on `curr == ClassNU || prev == ClassNU`

### Group 5: Closing Punctuation (HIGH HIT RATE: 5.24%)

#### LB13: × [CL CP EX IS SY]
- **Prev**: Any
- **Curr**: `ClassCL | ClassCP | ClassEX | ClassIS | ClassSY`
- **Logic**: `if curr in {CL, CP, EX, IS, SY} { return true, BreakNo }`
- **Bucket**: `curr in {ClassCL, ClassCP, ClassEX, ClassIS, ClassSY}`

### Group 6: Opening Punctuation

#### LB14: OP SP* ×
- **Prev**: `ClassOP` (may have SP between)
- **Curr**: Any (except certain classes)
- **Logic**: Scans backward past SP looking for OP
- **Bucket**: **Check when prev region has OP** (needs backward scan)

#### LB15: QU SP* × OP
- **Prev**: `ClassQU` (may have SP between)
- **Curr**: `ClassOP`
- **Logic**: `if curr == ClassOP` then scan backward past SP for QU
- **Bucket**: `curr == ClassOP` (then check for QU)

### Group 7: Quotation Marks (Complex - Multiple Variants)

#### LB19_Guillemet, LB19_German, LB19_QU_Pi_SP, etc.
- **Prev**: Various quotation classes `ClassQU, ClassQU_Pi, ClassQU_Pf`
- **Curr**: Various
- **Logic**: Complex context-sensitive patterns with backward scanning
- **Bucket**: **Complex** - may need `prev/curr in {QU, QU_Pi, QU_Pf, NS, ID}` OR always check

### Group 8: More Punctuation

#### LB16: (CL | CP) SP* × NS
- **Prev**: `ClassCL | ClassCP` (may have SP between)
- **Curr**: `ClassNS`
- **Logic**: `if curr == ClassNS` then scan backward past SP for CL/CP
- **Bucket**: `curr == ClassNS`

#### LB17: B2 SP* × B2
- **Prev**: `ClassB2` (may have SP between)
- **Curr**: `ClassB2`
- **Logic**: Scans backward past SP looking for B2
- **Bucket**: `curr == ClassB2 || prev == ClassB2`

#### LB20: ÷ CB, CB ÷
- **Prev**: `ClassCB` OR
- **Curr**: `ClassCB`
- **Logic**: `if curr == ClassCB || prev == ClassCB { return true, BreakYes }`
- **Bucket**: `prev == ClassCB || curr == ClassCB`

### Group 9: Hyphen Handling

#### LB21_HY, LB21_HY_SP_CM, LB21_HH_Break, LB21_HH
- **Prev**: Involves `ClassHL, ClassAL, ClassHY, ClassHH`
- **Curr**: Varies
- **Logic**: Complex Hebrew hyphen handling
- **Bucket**: `prev in {ClassHL, ClassAL, ClassHY, ClassHH}` OR `curr in {ClassHL, ClassHY, ClassHH}`

### Group 10: Alphabetics

#### LB22: AL × IN, HL × IN
- **Prev**: `ClassAL | ClassHL`
- **Curr**: `ClassIN`
- **Logic**: `if (prev == ClassAL || prev == ClassHL) && curr == ClassIN { return true, BreakNo }`
- **Bucket**: `curr == ClassIN && prev in {ClassAL, ClassHL}`

#### LB23: ID × PO, AL × NU, HL × NU
- **Prev**: `ClassID | ClassAL | ClassHL`
- **Curr**: `ClassPO | ClassNU`
- **Logic**: Multiple patterns
- **Bucket**: Complex

#### LB23a: PR × ID, PR × (AL | HL), PO × (AL | HL)
- **Prev**: `ClassPR | ClassPO`
- **Curr**: `ClassID | ClassAL | ClassHL`
- **Logic**: Multiple patterns
- **Bucket**: `prev in {ClassPR, ClassPO}`

#### LB24: (PR | PO) × (AL | HL)
- **Prev**: `ClassPR | ClassPO`
- **Curr**: `ClassAL | ClassHL`
- **Logic**: Pattern matching
- **Bucket**: `prev in {ClassPR, ClassPO} && curr in {ClassAL, ClassHL}`

### Group 11: Korean (Hangul)

#### LB26: JL × (JL | JV | H2 | H3), etc.
- **Prev**: Hangul classes `ClassJL | ClassJV | ClassJT | ClassH2 | ClassH3`
- **Curr**: Hangul classes
- **Logic**: Hangul syllable patterns
- **Bucket**: `prev in Hangul || curr in Hangul`

#### LB27: (JL | JV | JT | H2 | H3) × PO
- **Prev**: Hangul classes
- **Curr**: `ClassPO | ClassPR`
- **Logic**: Hangul with postfix
- **Bucket**: `prev in Hangul && curr in {ClassPO, ClassPR}`

### Group 12: Aksara/Indic

#### LB28 variants: AP, Virama, VI, etc.
- **Prev/Curr**: Aksara classes `ClassAK, ClassAP, ClassAS, ClassVI, ClassVF`
- **Logic**: Complex Indic script patterns
- **Bucket**: `prev in Aksara || curr in Aksara`

### Group 13: More Alphabetics

#### LB29: IS × (AL | HL)
- **Prev**: `ClassIS`
- **Curr**: `ClassAL | ClassHL`
- **Logic**: `if prev == ClassIS && (curr == ClassAL || curr == ClassHL) { return true, BreakNo }`
- **Bucket**: `prev == ClassIS && curr in {ClassAL, ClassHL}`

#### LB30: (AL | HL | NU) × OP
- **Prev**: `ClassAL | ClassHL | ClassNU`
- **Curr**: `ClassOP`
- **Logic**: Alphabetic/numeric before opening
- **Bucket**: `curr == ClassOP && prev in {ClassAL, ClassHL, ClassNU}`

### Group 14: Emoji

#### LB30a: RI × RI
- **Prev**: `ClassRI`
- **Curr**: `ClassRI`
- **Logic**: Regional indicator pairs
- **Bucket**: `prev == ClassRI && curr == ClassRI`

#### LB30b: EB × EM, ExtPict × EM
- **Prev**: `ClassEB` OR ExtPict property
- **Curr**: `ClassEM`
- **Logic**: Emoji base + modifier (has backward scan for ExtPict)
- **Bucket**: `curr == ClassEM`

#### LB31: ID_ExtPict × EM
- **Prev**: `ClassID` with ExtPict property
- **Curr**: `ClassEM`
- **Logic**: Ideograph emoji + modifier (has backward scan)
- **Bucket**: `curr == ClassEM` (combined with LB30b)

## Bucketing Strategy

### Fast-Path Buckets (Single Class Check)

1. **Curr-based buckets** (check `curr` first, fastest):
   ```go
   switch curr {
   case ClassZW:        // LB7
   case ClassWJ:        // LB11
   case ClassGL:        // LB12c, LB12a
   case ClassCL, ClassCP, ClassEX, ClassIS, ClassSY:  // LB13 (5.24% hit!)
   case ClassOP:        // LB15, LB30
   case ClassNS:        // LB16
   case ClassB2:        // LB17
   case ClassCB:        // LB20
   case ClassIN:        // LB22
   case ClassEM:        // LB30b, LB31
   // ... more
   }
   ```

2. **Prev-based buckets** (check `prev` second):
   ```go
   switch prev {
   case ClassBK:        // LB4
   case ClassCR:        // LB5a, LB5b
   case ClassLF, ClassNL:  // LB5b
   case ClassWJ:        // LB11
   case ClassGL:        // LB12
   case ClassBA:        // LB12c
   case ClassCB:        // LB20
   case ClassIS:        // LB29
   case ClassRI:        // LB30a
   // ... more
   }
   ```

### Complex Rules (Always Check)

These rules need backward scanning or complex logic:
- LB8a (checks rune, not class)
- LB8 (backward scan for ZW)
- LB14 (backward scan for OP past SP)
- LB19 variants (complex quotation patterns)
- LB21 variants (Hebrew hyphen logic)
- LB25 (numeric expression patterns)

### Three-Tier Strategy

**Tier 1: Curr-based dispatch** (fastest - single class check)
- Bucket rules by `curr` class
- ~50% of rules can be bucketed by `curr`

**Tier 2: Prev-based dispatch** (fast - single class check)
- Bucket rules by `prev` class
- Another ~30% of rules

**Tier 3: Complex rules** (slower - always check)
- Rules with backward scans or rune checks
- ~20% of rules

## Expected Performance Impact

**Current:** 38.5 rule checks per position (average)

**After bucketing:**
- Tier 1 (curr): ~5-10 rules checked per position
- Tier 2 (prev): ~3-7 rules checked per position
- Tier 3 (complex): ~8-10 rules checked per position
- **Total: ~15-25 checks per position**

**Estimated improvement: 1.5-2.5x reduction in rule checks**

This could bring Unicode text performance from **2.05x slower** → **1.0-1.5x slower** (near parity with original!).
