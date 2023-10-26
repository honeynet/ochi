package backend

import (
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
