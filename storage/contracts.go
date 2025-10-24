package storage

import (
	"commentservice/internal/models"
	"context"
	"log/slog"
)

type CommentStorage interface {
	AddComment(ctx context.Context, comment *models.Comment, log *slog.Logger) error
	GetComments(ctx context.Context, newsID, parrentID int, comment string, newsStore *newsStorage, log *slog.Logger) ([]*models.Comment, error)
	Close()
}
type NewsChecker interface {
	NewsExists(ctx context.Context, id int) (bool, error)
}
