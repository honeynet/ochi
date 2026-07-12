package backend

import (
	"bytes"
	"context"
	"strings"

	"github.com/honeynet/ochi/backend/handlers"
)

func isNotFoundError(e error) bool {
	return strings.Contains(e.Error(), "no rows in result set")
}

func userIDFromCtx(ctx context.Context) string {
	return ctx.Value(handlers.UserID("userID")).(string)
}

// IndexReplace substitutes the first occurrence of old with new in b.
func IndexReplace(b, old, new []byte) ([]byte, bool) {
	i := bytes.Index(b, old)
	if i == -1 {
		return b, false
	}
	copy(b[i:], new)
	return b, true
}
