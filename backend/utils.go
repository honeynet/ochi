package backend

import "strings"

func isNotFoundError(e error) bool {
	return strings.Contains(e.Error(), "no rows in result set")
}
