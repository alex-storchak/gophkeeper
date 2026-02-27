package ctxutil

import (
	"context"
	"fmt"

	"github.com/alex-storchak/gophkeeper/internal/models"
)

var (
	ErrUserNotFoundInContext       = fmt.Errorf("user not found in context")
	ErrUnexpectedUserTypeInContext = fmt.Errorf("unexpected user type in context")
)

type userCtxKey struct{}

func WithUser(ctx context.Context, userID models.UserID) context.Context {
	return context.WithValue(ctx, userCtxKey{}, userID)
}

func GetCtxUserID(ctx context.Context) (models.UserID, error) {
	value := ctx.Value(userCtxKey{})
	if value == nil {
		return 0, ErrUserNotFoundInContext
	}

	if userID, ok := value.(models.UserID); ok {
		return userID, nil
	}
	return 0, fmt.Errorf("%w: %T", ErrUnexpectedUserTypeInContext, value)
}
