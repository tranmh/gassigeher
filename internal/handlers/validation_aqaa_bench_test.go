package handlers

import (
	"fmt"
	"strings"
	"testing"
)

var benchSinkAqaa interface{}

var benchLengthsAqaa = []int{16, 1024, 65536}

func benchHTMLInputAqaa(n int) string {
	unit := `<b>Bello</b> <script>alert(1)</script> javascript:void(0) data:text/html `
	return strings.Repeat(unit, n/len(unit)+1)[:n]
}

func BenchmarkSanitizeString(b *testing.B) {
	for _, n := range benchLengthsAqaa {
		in := benchHTMLInputAqaa(n)
		b.Run(fmt.Sprintf("len=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchSinkAqaa = SanitizeString(in)
			}
		})
	}
}

func BenchmarkStripHTMLTags(b *testing.B) {
	for _, n := range benchLengthsAqaa {
		in := benchHTMLInputAqaa(n)
		b.Run(fmt.Sprintf("len=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchSinkAqaa = StripHTMLTags(in)
			}
		})
	}
}

func BenchmarkStripDangerousProtocols(b *testing.B) {
	for _, n := range benchLengthsAqaa {
		in := benchHTMLInputAqaa(n)
		b.Run(fmt.Sprintf("len=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchSinkAqaa = StripDangerousProtocols(in)
			}
		})
	}
}

func BenchmarkValidateDogExternalLink(b *testing.B) {
	for _, n := range []int{16, 256, 4096} {
		link := "https://example.org/" + strings.Repeat("a", n)
		b.Run(fmt.Sprintf("len=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				l := link
				v, err := ValidateDogExternalLink(&l)
				benchSinkAqaa = v
				benchSinkAqaa = err
			}
		})
	}
}

func BenchmarkValidateDogName(b *testing.B) {
	for _, n := range []int{8, 100, 4096} {
		name := benchHTMLInputAqaa(n)
		b.Run(fmt.Sprintf("len=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				v, err := ValidateDogName(name)
				benchSinkAqaa = v
				benchSinkAqaa = err
			}
		})
	}
}

func BenchmarkValidatePersonName(b *testing.B) {
	for _, n := range []int{8, 100, 4096} {
		name := benchHTMLInputAqaa(n)
		b.Run(fmt.Sprintf("len=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				v, err := ValidatePersonName(name, "first_name")
				benchSinkAqaa = v
				benchSinkAqaa = err
			}
		})
	}
}

func BenchmarkValidateOrganizationName(b *testing.B) {
	for _, n := range []int{8, 100, 4096} {
		name := benchHTMLInputAqaa(n)
		b.Run(fmt.Sprintf("len=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				v, err := ValidateOrganizationName(name)
				benchSinkAqaa = v
				benchSinkAqaa = err
			}
		})
	}
}
