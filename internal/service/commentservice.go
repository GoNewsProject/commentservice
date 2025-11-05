package service

import (
	"commentservice/internal/models"
	"commentservice/storage"
	"context"
	"fmt"
	"log/slog"
)

type CommentServiceImpl struct {
	commentsStorage storage.CommentsStorage
	newsStorage     storage.NewsStorage
	log             *slog.Logger
}

func NewCommentService(
	commentsStorage storage.CommentsStorage,
	newsStorage storage.NewsStorage,
	log *slog.Logger,
) CommentService {
	return &CommentServiceImpl{
		commentsStorage: commentsStorage,
		newsStorage:     newsStorage,
		log:             log,
	}
}

func (s *CommentServiceImpl) AddComment(ctx context.Context, newsID int, comment string) error {
	exists, err := s.newsStorage.NewsExists(ctx, newsID)
	if err != nil {
		s.log.Error("failed to check news existence", "news_id", newsID, "error", err)
		return fmt.Errorf("failed to check news existence: %w", err)
	}
	if !exists {
		s.log.Warn("news not found", "news_id", newsID)
		return fmt.Errorf("news with id %d not found", newsID)
	}

	if err := s.commentsStorage.AddComment(ctx, newsID, comment); err != nil {
		s.log.Error("failed to save comment", "news_id", newsID, "error", err)
		return fmt.Errorf("failed to save comment: %w", err)
	}

	s.log.Info("comment added successfully", "news_id", newsID)
	return nil
}

func (s *CommentServiceImpl) GetComments(ctx context.Context, newsID int) ([]models.Comment, error) {
	exists, err := s.newsStorage.NewsExists(ctx, newsID)
	if err != nil {
		s.log.Error("failed to check news existance in database")
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("news with id %d not found", newsID)
	}
	comments, err := s.commentsStorage.GetComments(ctx, newsID)
	if err != nil {
		s.log.Error("failed to get comments from database")
		return nil, err
	}

	return comments, nil
}
