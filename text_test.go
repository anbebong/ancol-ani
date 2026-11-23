package main

import (
	"strings"
	"testing"
)

func TestNormalizeText(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Hello", "Hello"},
		{"  Hello  ", "Hello"},
		{"Chào buổi sáng!\nĐang họp nhé", "Chào buổi sáng!\nĐang họp nhé"},
		{"a\r\n\r\n  b  \n\n\nc\n", "a\nb\nc"}, // CRLF, blank lines, padding
		{"\n \n\t\n", ""},
	}
	for _, c := range cases {
		if got := normalizeText(c.in); got != c.want {
			t.Errorf("normalizeText(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	// at most maxTextLines lines
	many := strings.Repeat("x\n", maxTextLines+5)
	if n := len(strings.Split(normalizeText(many), "\n")); n != maxTextLines {
		t.Errorf("lines = %d, want %d", n, maxTextLines)
	}

	// each line cut at maxTextLineRunes runes (not bytes - Vietnamese is multi-byte)
	long := strings.Repeat("ạ", maxTextLineRunes+10)
	if n := len([]rune(normalizeText(long))); n != maxTextLineRunes {
		t.Errorf("runes = %d, want %d", n, maxTextLineRunes)
	}
}
