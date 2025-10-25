package storage

import (
	"commentservice/internal/models"
	"context"
)

type CommentStorage interface {
	AddComment(ctx context.Context, comment *models.Comment) error
	GetComments(ctx context.Context, newsID string, limit int, offset int) ([]*models.Comment, error)
	CheckCommentIDExists(ctx context.Context, commentID int) bool
	Close()
}
type NewsChecker interface {
	NewsExists(ctx context.Context, id int) (bool, error)
}
