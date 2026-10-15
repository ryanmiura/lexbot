package service

import "strings"

// NormalizeListFilter translates supported PT/EN arguments to repository keys.
// Unknown arguments are rejected so a typo never silently lists every word.
func NormalizeListFilter(filter string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(filter)) {
	case "":
		return "", nil
	case "novas", "nova", "new":
		return "novas", nil
	case "dificeis", "difíceis", "dificil", "difícil", "difficult":
		return "dificeis", nil
	default:
		return "", ErrInvalidListFilter
	}
}
