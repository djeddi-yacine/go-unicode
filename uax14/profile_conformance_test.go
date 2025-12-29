package uax14

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// Rule profiling counters
var (
	ruleMatchCount = make(map[int]int64)  // rule index -> match count
	ruleCheckCount = make(map[int]int64)  // rule index -> check count
	pairTableCount int64
	totalPositions int64
)

func resetProfile() {
	ruleMatchCount = make(map[int]int64)
	ruleCheckCount = make(map[int]int64)
	pairTableCount = 0
	totalPositions = 0
}

// TestProfileConformanceTests profiles all official conformance tests
func TestProfileConformanceTests(t *testing.T) {
	resetProfile()

	// Try to find the test file in various locations
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

		// Build test string and expected breaks
		var runes []rune
		var expectedBreaks []bool

		for i := 0; i < len(tokens); i++ {
			token := tokens[i]

			if token == "÷" {
				if len(runes) > 0 {
					expectedBreaks[len(expectedBreaks)-1] = true
				}
			} else if token == "×" {
				// No break
			} else {
				// Parse hex codepoint
				var codepoint rune
				if _, err := fmt.Sscanf(token, "%X", &codepoint); err == nil {
					runes = append(runes, codepoint)
					expectedBreaks = append(expectedBreaks, false)
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
		for ctx.Slide() {
			totalPositions++

			// Check each rule in order
			matched := false
			for idx, rule := range lineBreakRules {
				ruleCheckCount[idx]++

				m, _ := rule(ctx)
				if m {
					ruleMatchCount[idx]++
					matched = true
					break
				}
			}

			// Pair table fallback
			if !matched {
				pairTableCount++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading test file: %v", err)
	}

	t.Logf("Profiled %d test cases, %d total positions", testCount, totalPositions)

	// Print report
	printProfile(t)
}

func printProfile(t *testing.T) {
	type ruleStat struct {
		index        int
		name         string
		matches      int64
		checks       int64
		matchPercent float64
		hitRate      float64
		cumulative   float64
	}

	ruleNames := []string{
		"LB4", "LB5a", "LB5b", "LB7", "LB8a", "LB8",
		"LB11", "LB12", "LB12c", "LB12a",
		"LB25", "LB13",
		"LB19_Guillemet", "LB19_German", "LB19_QU_Pi_SP",
		"LB14", "LB15", "LB16", "LB17",
		"LB19_NS_QU_Pi", "LB19_CJK_QU_Pf_ID", "LB19_CJK_ID_QU_Pi", "LB19_SP_QU_Pf",
		"LB20",
		"LB21_HY", "LB21_HY_SP_CM", "LB21_HH_Break", "LB21_HH",
		"LB22", "LB23", "LB23a", "LB24",
		"LB25a", "LB25b", "LB25c", "LB25d", "LB25e",
		"LB26", "LB27", "LB28", "LB29", "LB30", "LB30a", "LB30b",
		"LB31_AK_AP", "LB31_AK_AS", "LB31_AP_AK", "LB31_AS_AK",
		"LB31_AS_VI", "LB31_VI_AK", "LB31_VI_AS", "LB31_VF",
	}

	var stats []ruleStat
	totalMatches := pairTableCount

	for _, matches := range ruleMatchCount {
		totalMatches += matches
	}

	for idx, matches := range ruleMatchCount {
		checks := ruleCheckCount[idx]
		var name string
		if idx < len(ruleNames) {
			name = ruleNames[idx]
		} else {
			name = fmt.Sprintf("Rule%d", idx)
		}

		stats = append(stats, ruleStat{
			index:        idx,
			name:         name,
			matches:      matches,
			checks:       checks,
			matchPercent: float64(matches) / float64(totalMatches) * 100,
			hitRate:      float64(matches) / float64(checks) * 100,
		})
	}

	// Add pair table
	stats = append(stats, ruleStat{
		index:        -1,
		name:         "PairTable",
		matches:      pairTableCount,
		checks:       pairTableCount, // Always "checked" when we get to it
		matchPercent: float64(pairTableCount) / float64(totalMatches) * 100,
		hitRate:      100.0,
	})

	// Sort by match count
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].matches > stats[j].matches
	})

	// Calculate cumulative
	var cumulative float64
	for i := range stats {
		cumulative += stats[i].matchPercent
		stats[i].cumulative = cumulative
	}

	// Calculate average checks per position
	totalChecks := int64(0)
	for _, checks := range ruleCheckCount {
		totalChecks += checks
	}
	avgChecks := float64(totalChecks) / float64(totalPositions)

	t.Logf("\n=== UAX #14 Rule Match Profile (Conformance Tests) ===")
	t.Logf("Total positions checked: %d", totalPositions)
	t.Logf("Total rule matches: %d", totalMatches)
	t.Logf("Average rules checked per position: %.2f", avgChecks)
	t.Logf("")
	t.Logf("Top 20 rules by match frequency:")
	t.Logf("%-4s | %-22s | %10s | %9s | %10s | %8s", "Rank", "Rule", "Matches", "% Total", "Cumulative", "Hit Rate")
	t.Logf("%s", strings.Repeat("-", 80))

	for i, s := range stats {
		if i >= 20 {
			break
		}
		t.Logf("%4d | %-22s | %10d | %8.2f%% | %9.2f%% | %7.2f%%",
			i+1, s.name, s.matches, s.matchPercent, s.cumulative, s.hitRate)
	}

	// Find coverage thresholds
	t.Logf("")
	for i, s := range stats {
		if s.cumulative >= 80.0 {
			t.Logf("Top %d rules cover 80%% of matches", i+1)
			break
		}
	}

	for i, s := range stats {
		if s.cumulative >= 90.0 {
			t.Logf("Top %d rules cover 90%% of matches", i+1)
			break
		}
	}

	// Show which rules are rarely used
	t.Logf("\nRarely matched rules (< 0.1%%):")
	for _, s := range stats {
		if s.matchPercent < 0.1 && s.matchPercent > 0 {
			t.Logf("  %s: %.3f%% (%d matches)", s.name, s.matchPercent, s.matches)
		}
	}
}
