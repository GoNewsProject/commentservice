package service

import (
	"commentservice/internal/models"
	"context"
)

type CommentService interface {
	AddComment(ctx context.Context, newsID int, comment string) error
	GetComments(ctx context.Context, newsID int) ([]models.Comment, error)
}
