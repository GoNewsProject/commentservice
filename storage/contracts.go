package storage

import (
	"commentservice/internal/models"
	"context"
)

type CommentStorage interface {
	AddComment(ctx context.Context, newsID int, comment string) error
	GetComments(ctx context.Context, newsID int) ([]models.Comment, error)
	Close()
}
type NewsChecker interface {
	NewsExists(ctx context.Context, newsID int) (bool, error)
}
