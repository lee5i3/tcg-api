package catalog

import (
	"errors"
	"testing"
)

func TestNormLanguage(t *testing.T) {
	for _, tc := range []struct {
		in      *string
		want    string
		wantErr bool
	}{
		{nil, "eng", false},
		{ptr(""), "eng", false},
		{ptr("  "), "eng", false},
		{ptr("jpn"), "jpn", false},
		{ptr(" ENG "), "eng", false},
		{ptr("en"), "", true},
		{ptr("english"), "", true},
		{ptr("e n"), "", true},
	} {
		got, err := normLanguage(tc.in)
		if tc.wantErr {
			if !errors.Is(err, ErrInvalid) {
				t.Errorf("normLanguage(%v) err = %v, want ErrInvalid", tc.in, err)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("normLanguage(%v) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
}

func TestNormReleaseDate(t *testing.T) {
	if got, err := normReleaseDate(nil); got != nil || err != nil {
		t.Errorf("nil date: %v, %v", got, err)
	}
	if got, err := normReleaseDate(ptr("2023-09-22")); err != nil || *got != "2023-09-22" {
		t.Errorf("dash date: %v, %v", got, err)
	}
	if got, err := normReleaseDate(ptr("2023/09/22")); err != nil || *got != "2023-09-22" {
		t.Errorf("slash date normalized: %v, %v", got, err)
	}
	if _, err := normReleaseDate(ptr("Sept 22, 2023")); !errors.Is(err, ErrInvalid) {
		t.Errorf("bad date err = %v, want ErrInvalid", err)
	}
}

func TestNumberSortKey(t *testing.T) {
	for _, tc := range []struct {
		in   string
		n    int
		tail string
	}{
		{"145", 145, "145"},
		{"1a", 1, "1a"},
		{"GG01", 0, "GG01"},
		{"", 0, ""},
	} {
		n, tail := numberSortKey(tc.in)
		if n != tc.n || tail != tc.tail {
			t.Errorf("numberSortKey(%q) = %d, %q; want %d, %q", tc.in, n, tail, tc.n, tc.tail)
		}
	}
}

func TestItoa(t *testing.T) {
	for _, tc := range []struct {
		in   int
		want string
	}{{0, "0"}, {7, "7"}, {501773, "501773"}, {-3, "-3"}} {
		if got := itoa(tc.in); got != tc.want {
			t.Errorf("itoa(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
