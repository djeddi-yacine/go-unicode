package uax14

import (
	"testing"
)

// BenchmarkASCIIFastPath benchmarks pure ASCII text that takes the fast path
func BenchmarkASCIIFastPath(b *testing.B) {
	tests := []struct {
		name string
		text string
	}{
		{
			name: "Short",
			text: "hello world",
		},
		{
			name: "Medium",
			text: "The quick brown fox jumps over the lazy dog and runs away quickly",
		},
		{
			name: "Long",
			text: "This is a longer piece of English text with multiple words and spaces that should demonstrate the performance improvement of the ASCII fast path optimization over the full Unicode line breaking algorithm implementation",
		},
		{
			name: "WithNewlines",
			text: "Line one\nLine two\nLine three\nLine four\nLine five\n",
		},
		{
			name: "Code",
			text: "func main() {\n\tif x == y {\n\t\treturn true\n\t}\n\treturn false\n}\n",
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = FindLineBreakOpportunities(tt.text, HyphensNone)
			}
		})
	}
}

// BenchmarkUnicodePath benchmarks text with Unicode that skips the fast path
func BenchmarkUnicodePath(b *testing.B) {
	tests := []struct {
		name string
		text string
	}{
		{
			name: "WithEmoji",
			text: "Hello 👋 world 🌍",
		},
		{
			name: "Mixed",
			text: "This is English text with some 中文 characters mixed in",
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = FindLineBreakOpportunities(tt.text, HyphensNone)
			}
		})
	}
}
