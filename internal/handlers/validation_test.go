package handlers

import (
	"strings"
	"testing"
)

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"plain text unchanged", "Hello", "Hello"},
		{"keeps text between tags", "<b>bold</b> text", "bold text"},
		{"script tag removed, content kept", "<script>alert(1)</script>", "alert(1)"},
		{"img with attributes", `<img src=x onerror="alert(1)">Rex`, "Rex"},
		{"empty tag", "a<>b", "ab"},
		{"unclosed bracket kept", "a < b", "a < b"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripHTMLTags(tt.input); got != tt.want {
				t.Errorf("StripHTMLTags(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStripDangerousProtocols(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"javascript", "javascript:alert(1)", ""},
		{"upper case", "JAVASCRIPT:alert(1)", ""},
		{"leading whitespace", " \t javascript:alert(1)", ""},
		{"space before colon", "javascript :alert(1)", ""},
		{"data", "data:text/html;base64,xx", ""},
		{"vbscript", "VbScript:msgbox(1)", ""},
		{"file", "file:///etc/passwd", ""},
		{"decimal entity colon", "javascript&#58;alert(1)", ""},
		{"decimal entity zero padded", "javascript&#0058;alert(1)", ""},
		{"hex entity colon", "javascript&#x3a;alert(1)", ""},
		{"hex entity upper", "javascript&#X3A;alert(1)", ""},
		{"https unchanged", "https://example.com", "https://example.com"},
		{"protocol not at start unchanged", "see javascript:alert(1)", "see javascript:alert(1)"},
		{"word without colon unchanged", "javascript is fun", "javascript is fun"},
		{"whitespace preserved when safe", "  hello  ", "  hello  "},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripDangerousProtocols(tt.input); got != tt.want {
				t.Errorf("StripDangerousProtocols(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"trims whitespace", "  Rex \n", "Rex"},
		{"strips tags and trims", "  <i>Bello</i>  ", "Bello"},
		{"only tags becomes empty", "<script></script>", ""},
		{"tag hiding javascript protocol", "<b></b>javascript:alert(1)", ""},
		{"protocol after tag with whitespace", "<p> javascript:alert(1)</p>", ""},
		{"umlauts kept", "Müller & Söhne", "Müller & Söhne"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeString(tt.input); got != tt.want {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeStringPtr(t *testing.T) {
	if got := SanitizeStringPtr(nil); got != nil {
		t.Errorf("SanitizeStringPtr(nil) = %q, want nil", *got)
	}
	in := " <b>Rex</b> "
	got := SanitizeStringPtr(&in)
	if got == nil || *got != "Rex" {
		t.Fatalf("SanitizeStringPtr = %v, want pointer to \"Rex\"", got)
	}
	if in != " <b>Rex</b> " {
		t.Errorf("input was modified: %q", in)
	}
}

func TestValidateDogName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantMsg string
	}{
		{"valid", "Rex", "Rex", ""},
		{"sanitized", " <b>Rex</b> ", "Rex", ""},
		{"empty", "", "", "Name is required"},
		{"whitespace only", "   ", "", "Name is required"},
		{"only tags", "<script></script>", "", "Name is required"},
		{"exactly 100", strings.Repeat("a", 100), strings.Repeat("a", 100), ""},
		{"101 chars", strings.Repeat("a", 101), "", "Name must be 100 characters or less"},
		{"101 chars incl tags is fine after sanitizing", "<b>" + strings.Repeat("a", 100) + "</b>", strings.Repeat("a", 100), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, verr := ValidateDogName(tt.input)
			if tt.wantMsg == "" {
				if verr != nil {
					t.Fatalf("unexpected error: %+v", verr)
				}
				if got != tt.want {
					t.Errorf("got %q, want %q", got, tt.want)
				}
				return
			}
			if verr == nil {
				t.Fatalf("expected error %q, got value %q", tt.wantMsg, got)
			}
			if verr.Field != "name" || verr.Message != tt.wantMsg {
				t.Errorf("error = %+v, want field name / %q", verr, tt.wantMsg)
			}
			if got != "" {
				t.Errorf("value on error = %q, want empty", got)
			}
		})
	}
}

func TestValidatePersonName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantMsg string
	}{
		{"valid", " Anna ", "Anna", ""},
		{"empty", "", "", "first_name ist erforderlich"},
		{"only tags", "<i></i>", "", "first_name ist erforderlich"},
		{"exactly 100", strings.Repeat("b", 100), strings.Repeat("b", 100), ""},
		{"101", strings.Repeat("b", 101), "", "first_name darf maximal 100 Zeichen haben"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, verr := ValidatePersonName(tt.input, "first_name")
			if tt.wantMsg == "" {
				if verr != nil || got != tt.want {
					t.Errorf("got (%q, %+v), want (%q, nil)", got, verr, tt.want)
				}
				return
			}
			if verr == nil || verr.Field != "first_name" || verr.Message != tt.wantMsg {
				t.Errorf("error = %+v, want field first_name / %q", verr, tt.wantMsg)
			}
		})
	}
}

func TestValidateOrganizationName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantMsg string
	}{
		{"valid", "<b>Tierheim</b> Stuttgart ", "Tierheim Stuttgart", ""},
		{"empty", "  ", "", "Organisationsname ist erforderlich"},
		{"exactly 200", strings.Repeat("c", 200), strings.Repeat("c", 200), ""},
		{"201", strings.Repeat("c", 201), "", "Organisationsname darf maximal 200 Zeichen haben"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, verr := ValidateOrganizationName(tt.input)
			if tt.wantMsg == "" {
				if verr != nil || got != tt.want {
					t.Errorf("got (%q, %+v), want (%q, nil)", got, verr, tt.want)
				}
				return
			}
			if verr == nil || verr.Field != "organization_name" || verr.Message != tt.wantMsg {
				t.Errorf("error = %+v, want field organization_name / %q", verr, tt.wantMsg)
			}
		})
	}
}

func TestValidateDogExternalLink(t *testing.T) {
	strp := func(s string) *string { return &s }
	long := func(n int) string { return "https://" + strings.Repeat("a", n-len("https://")) }

	tests := []struct {
		name    string
		input   *string
		want    *string
		wantMsg string
	}{
		{"nil", nil, nil, ""},
		{"blank", strp(""), nil, ""},
		{"whitespace only", strp("   "), nil, ""},
		{"valid https", strp(" https://example.com/dog "), strp("https://example.com/dog"), ""},
		{"javascript", strp("javascript:alert(1)"), nil, "Invalid URL protocol"},
		{"javascript mixed case padded", strp("  JavaScript:alert(1)"), nil, "Invalid URL protocol"},
		{"data", strp("data:text/html,x"), nil, "Invalid URL protocol"},
		{"vbscript", strp("VBSCRIPT:x"), nil, "Invalid URL protocol"},
		{"file", strp("file:///etc/passwd"), nil, "Invalid URL protocol"},
		{"only script tags", strp("<script></script>"), nil, "Invalid URL content"},
		{"entity encoded javascript", strp("javascript&#58;alert(1)"), nil, "Invalid URL content"},
		{"exactly 500", strp(long(500)), strp(long(500)), ""},
		{"501", strp(long(501)), nil, "External link must be 500 characters or less"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, verr := ValidateDogExternalLink(tt.input)
			if tt.wantMsg == "" {
				if verr != nil {
					t.Fatalf("unexpected error: %+v", verr)
				}
				if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
					t.Errorf("got %v, want %v", got, tt.want)
				}
				return
			}
			if verr == nil {
				t.Fatalf("expected error %q, got %v", tt.wantMsg, got)
			}
			if verr.Field != "external_link" || verr.Message != tt.wantMsg {
				t.Errorf("error = %+v, want field external_link / %q", verr, tt.wantMsg)
			}
			if got != nil {
				t.Errorf("value on error = %q, want nil", *got)
			}
		})
	}
}
