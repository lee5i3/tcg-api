package catalog

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var gameKeyRe = regexp.MustCompile(`^[a-z0-9-]{2,32}$`)

var languageRe = regexp.MustCompile(`^[a-z]{3}$`)

// normLanguage validates an ISO 639 alpha-3 language reference.
// A nil/blank value defaults to "eng".
func normLanguage(raw *string) (string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return "eng", nil
	}
	lang := strings.ToLower(strings.TrimSpace(*raw))
	if !languageRe.MatchString(lang) {
		return "", fmt.Errorf("%w: language must be an ISO 639 alpha-3 code (e.g. \"eng\", \"jpn\")", ErrInvalid)
	}
	return lang, nil
}

// normReleaseDate accepts YYYY-MM-DD (or the source-style YYYY/MM/DD) and
// normalizes to YYYY-MM-DD. Nil/blank stays nil.
func normReleaseDate(raw *string) (*string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	for _, layout := range []string{"2006-01-02", "2006/01/02"} {
		if t, err := time.Parse(layout, strings.TrimSpace(*raw)); err == nil {
			formatted := t.Format("2006-01-02")
			return &formatted, nil
		}
	}
	return nil, fmt.Errorf("%w: releaseDate must be YYYY-MM-DD", ErrInvalid)
}

var leadingDigits = regexp.MustCompile(`^[0-9]+`)

// numberSortKey mirrors the old SQL ordering: leading digits numerically
// (missing → 0), then the raw string.
func numberSortKey(number string) (int, string) {
	n := 0
	if m := leadingDigits.FindString(number); m != "" {
		if v, err := strconv.Atoi(m); err == nil {
			n = v
		}
	}
	return n, number
}
