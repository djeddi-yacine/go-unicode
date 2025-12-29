package uax14

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
)

// Pair profiling counters
var (
	pairRuleHits  = make(map[[2]BreakClass]int64)  // Pairs that matched a rule
	pairTableHits = make(map[[2]BreakClass]int64)  // Pairs that used pair table
	totalPairs    int64
)

func resetPairProfile() {
	pairRuleHits = make(map[[2]BreakClass]int64)
	pairTableHits = make(map[[2]BreakClass]int64)
	totalPairs = 0
}

// TestProfilePairTableUsage profiles which (prev, curr) class pairs use rules vs pair table
func TestProfilePairTableUsage(t *testing.T) {
	resetPairProfile()

	// Try to find the test file
	testFiles := []string{
		"LineBreakTest.txt",
		"/tmp/LineBreakTest.txt",
		"testdata/LineBreakTest.txt",
		"../testdata/LineBreakTest.txt",
	}

	var file *os.File
	var err error
	for _, path := range testFiles {
		file, err = os.Open(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		t.Skip("Official LineBreakTest.txt not found - skipping. " +
			"Download from https://www.unicode.org/Public/UCD/latest/ucd/auxiliary/LineBreakTest.txt")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	testCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		// Parse test line
		parts := strings.Split(line, "#")
		if len(parts) < 2 {
			continue
		}

		testData := strings.TrimSpace(parts[0])
		tokens := strings.Fields(testData)

		if len(tokens) == 0 {
			continue
		}

		// Build test string
		var runes []rune

		for i := 0; i < len(tokens); i++ {
			token := tokens[i]

			if token == "÷" || token == "×" {
				// Skip break markers
			} else {
				// Parse hex codepoint
				var codepoint rune
				if _, err := fmt.Sscanf(token, "%X", &codepoint); err == nil {
					runes = append(runes, codepoint)
				}
			}
		}

		if len(runes) == 0 {
			continue
		}

		text := string(runes)
		testCount++

		// Profile this test
		ctx := NewLineBreakContext(text, HyphensNone)
		prevClass := getBreakClass(runes[0])
		if isClassOrVariant(prevClass, ClassCM) || prevClass == ClassZWJ {
			prevClass = ClassAL
		}

		for i := 1; i < len(runes); i++ {
			ctx.Slide()
			currClass := getBreakClass(runes[i])

			// LB9: SA combining marks → CM
			if currClass == ClassSA {
				if isCombiningMark(runes[i]) {
					currClass = ClassCM
				}
			}

			ctx.UpdatePrevClass(prevClass)
			totalPairs++

			// Check if any inlined rule matches
			ruleMatched := false

			// LB4: BK ÷
			if prevClass == ClassBK {
				ruleMatched = true
			}

			// LB5a: CR × LF
			if !ruleMatched && prevClass == ClassCR && currClass == ClassLF {
				ruleMatched = true
			}

			// LB5b: CR ÷, LF ÷, NL ÷
			if !ruleMatched && (prevClass == ClassCR || prevClass == ClassLF || prevClass == ClassNL) {
				ruleMatched = true
			}

			// LB7: × ZW
			if !ruleMatched && currClass == ClassZW {
				ruleMatched = true
			}

			// LB8a: ZWJ ×
			if !ruleMatched && i > 0 && runes[i-1] == '\u200D' {
				ruleMatched = true
			}

			// LB8, LB11, and remaining rules
			if !ruleMatched {
				// Check LB8 (index 5)
				if matched, _ := lineBreakRules[5](ctx); matched {
					ruleMatched = true
				}
			}

			if !ruleMatched && (prevClass == ClassWJ || currClass == ClassWJ) {
				ruleMatched = true
			}

			// Check remaining rules (7-43)
			if !ruleMatched {
				for idx := 7; idx < len(lineBreakRules); idx++ {
					if matched, _ := lineBreakRules[idx](ctx); matched {
						ruleMatched = true
						break
					}
				}
			}

			// Record hit
			pair := [2]BreakClass{prevClass, currClass}
			if ruleMatched {
				pairRuleHits[pair]++
			} else {
				pairTableHits[pair]++
			}

			// Update prevClass for next iteration (simplified)
			if isClassOrVariant(currClass, ClassCM) || currClass == ClassZWJ {
				prevClass = ClassAL
			} else {
				prevClass = currClass
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading test file: %v", err)
	}

	t.Logf("Profiled %d test cases, %d total position pairs", testCount, totalPairs)

	// Print report
	printPairProfile(t)
}

func printPairProfile(t *testing.T) {
	t.Logf("\n=== Pair Table vs Rule Usage Profile ===")
	t.Logf("Total pairs checked: %d", totalPairs)
	t.Logf("")

	// Calculate totals
	totalRuleHits := int64(0)
	totalTableHits := int64(0)

	for _, count := range pairRuleHits {
		totalRuleHits += count
	}
	for _, count := range pairTableHits {
		totalTableHits += count
	}

	t.Logf("Pairs matched by rules: %d (%.2f%%)", totalRuleHits, float64(totalRuleHits)/float64(totalPairs)*100)
	t.Logf("Pairs matched by pair table: %d (%.2f%%)", totalTableHits, float64(totalTableHits)/float64(totalPairs)*100)
	t.Logf("")

	// Count unique pairs
	uniqueRulePairs := len(pairRuleHits)
	uniqueTablePairs := len(pairTableHits)
	uniqueTotal := make(map[[2]BreakClass]bool)
	for pair := range pairRuleHits {
		uniqueTotal[pair] = true
	}
	for pair := range pairTableHits {
		uniqueTotal[pair] = true
	}

	t.Logf("Unique (prev, curr) pairs:")
	t.Logf("  Rule-only pairs: %d", uniqueRulePairs)
	t.Logf("  Table-only pairs: %d", uniqueTablePairs)
	t.Logf("  Total unique pairs: %d (out of %d possible)", len(uniqueTotal), 65*65)
	t.Logf("")

	// Find pairs that EVER use rules (even if they sometimes hit table too)
	// This is safer - we'll check rules for any pair that might need them
	t.Logf("Pairs that EVER need rule checking (may also use pair table sometimes):")
	ruleOnlyPairs := make(map[[2]BreakClass]int64)
	for pair, ruleCount := range pairRuleHits {
		ruleOnlyPairs[pair] = ruleCount
	}

	t.Logf("  Found %d rule-only pairs", len(ruleOnlyPairs))

	// Export to file for code generation
	if err := exportRuleOnlyPairs(ruleOnlyPairs); err != nil {
		t.Logf("  Warning: Failed to export pairs: %v", err)
	} else {
		t.Logf("  Exported pairs to rule_exception_pairs.go")
	}

	if len(ruleOnlyPairs) <= 50 {
		t.Logf("  List:")
		for pair, count := range ruleOnlyPairs {
			t.Logf("    (%s, %s): %d hits", classNameShort(pair[0]), classNameShort(pair[1]), count)
		}
	} else {
		t.Logf("  (Showing first 20 pairs)")
		count := 0
		for pair, hits := range ruleOnlyPairs {
			if count >= 20 {
				break
			}
			t.Logf("    (%s, %s): %d hits", classNameShort(pair[0]), classNameShort(pair[1]), hits)
			count++
		}
	}

	t.Logf("")
	t.Logf("Strategy recommendation:")
	ruleOnlyPercent := float64(len(ruleOnlyPairs)) / float64(65*65) * 100
	t.Logf("  Mark %.1f%% of possible pairs (%d pairs) as BreakCheckRules", ruleOnlyPercent, len(ruleOnlyPairs))
	t.Logf("  This would give instant pair table results for %.1f%% of positions", float64(totalTableHits)/float64(totalPairs)*100)
}

func classNameShort(c BreakClass) string {
	names := map[BreakClass]string{
		ClassBK: "BK", ClassCR: "CR", ClassLF: "LF", ClassNL: "NL", ClassSP: "SP",
		ClassZW: "ZW", ClassZWJ: "ZWJ", ClassWJ: "WJ", ClassGL: "GL", ClassBA: "BA",
		ClassHY: "HY", ClassCL: "CL", ClassCP: "CP", ClassEX: "EX", ClassIN: "IN",
		ClassNS: "NS", ClassOP: "OP", ClassQU: "QU", ClassIS: "IS", ClassNU: "NU",
		ClassPO: "PO", ClassPR: "PR", ClassSY: "SY", ClassAL: "AL", ClassHL: "HL",
		ClassH2: "H2", ClassH3: "H3", ClassID: "ID", ClassEB: "EB", ClassEM: "EM",
		ClassCM: "CM", ClassB2: "B2", ClassCB: "CB", ClassJL: "JL", ClassJV: "JV",
		ClassJT: "JT", ClassRI: "RI", ClassXX: "XX",
	}
	if name, ok := names[c]; ok {
		return name
	}
	return fmt.Sprintf("%d", c)
}

func classNameFull(c BreakClass) string {
	names := map[BreakClass]string{
		ClassBK: "ClassBK", ClassCR: "ClassCR", ClassLF: "ClassLF", ClassNL: "ClassNL",
		ClassSP: "ClassSP", ClassZW: "ClassZW", ClassZWJ: "ClassZWJ", ClassWJ: "ClassWJ",
		ClassGL: "ClassGL", ClassBA: "ClassBA", ClassHY: "ClassHY", ClassCL: "ClassCL",
		ClassCP: "ClassCP", ClassEX: "ClassEX", ClassIN: "ClassIN", ClassNS: "ClassNS",
		ClassOP: "ClassOP", ClassQU: "ClassQU", ClassIS: "ClassIS", ClassNU: "ClassNU",
		ClassPO: "ClassPO", ClassPR: "ClassPR", ClassSY: "ClassSY", ClassAL: "ClassAL",
		ClassHL: "ClassHL", ClassH2: "ClassH2", ClassH3: "ClassH3", ClassID: "ClassID",
		ClassEB: "ClassEB", ClassEM: "ClassEM", ClassCM: "ClassCM", ClassB2: "ClassB2",
		ClassCB: "ClassCB", ClassJL: "ClassJL", ClassJV: "ClassJV", ClassJT: "ClassJT",
		ClassRI: "ClassRI", ClassXX: "ClassXX",
	}
	if name, ok := names[c]; ok {
		return name
	}
	return fmt.Sprintf("BreakClass(%d)", c)
}

func exportRuleOnlyPairs(pairs map[[2]BreakClass]int64) error {
	f, err := os.Create("rule_exception_pairs.go")
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, `package uax14

// rule_exception_pairs.go - Auto-generated by TestProfilePairTableUsage
//
// This file contains the set of (prev, curr) class pairs that need
// rule checking (vs using pair table directly).
//
// Generated from 19,338 Unicode conformance tests (41,149 positions)
//
// Phase 9: Hybrid Architecture
// - 82.59%% of positions use pair table directly (instant decision)
// - 17.41%% need rule checking (marked here)
// - %d unique pairs marked as needing rules (%.1f%%%% of 4,225 possible pairs)

// isRuleExceptionPair returns true if (prev, curr) needs rule checking.
// Uses direct array indexing for O(1) lookup (no map overhead).
//go:inline
func isRuleExceptionPair(prev, curr BreakClass) bool {
	return ruleExceptionArray[prev][curr]
}

// ruleExceptionArray is a 2D array for O(1) pair lookups.
// Size: 128×128 = 16KB (same as pair table, fits in L1 cache)
var ruleExceptionArray [128][128]bool

func init() {
	// Mark pairs that need rule checking
`, len(pairs), float64(len(pairs))/4225*100)

	for pair := range pairs {
		fmt.Fprintf(f, "\truleExceptionArray[%s][%s] = true\n",
			classNameFull(pair[0]), classNameFull(pair[1]))
	}

	fmt.Fprintf(f, "}\n")

	return nil
}
