package handlers

import "testing"

func TestValidEntityType(t *testing.T) {
	valid := []string{"note", "todo", "memo", "bookmark", "clipboard"}
	for _, et := range valid {
		if !validEntityType(et) {
			t.Errorf("expected %q to be valid", et)
		}
	}
}

func TestInvalidEntityType(t *testing.T) {
	invalid := []string{"", "calendar", "event", "contact", "NOTE", "Todo", "MEMO", "note\x00"}
	for _, et := range invalid {
		if validEntityType(et) {
			t.Errorf("expected %q to be invalid", et)
		}
	}
}

func TestSanitizeFTSQuery(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"hello", `"hello"*`},
		{"with spaces", `"with spaces"*`},
		{`has "quotes"`, `"has ""quotes"""*`},
		{"", `""*`},
	}
	for _, c := range cases {
		got := sanitizeFTSQuery(c.input)
		if got != c.want {
			t.Errorf("sanitizeFTSQuery(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
