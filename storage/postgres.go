package storage

import (
	"commentservice/internal/models"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
)

type Storage struct {
	db          *pgxpool.Pool
	log         *slog.Logger
	newsChecker NewsChecker
}

// NewsAPIClient для проверки новостей через API
type NewsAPIClient struct {
	BaseURL string
	Client  *http.Client
	log     *slog.Logger
}

func NewNewsAPIClient(baseURL string, log *slog.Logger) *NewsAPIClient {
	return &NewsAPIClient{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 10 * time.Second},
		log:     log,
	}
}

func NewCommentStorage(ctx context.Context, dbName string, log *slog.Logger, newsChecker NewsChecker) (*Storage, error) {
	err := godotenv.Load()
	if err != nil {
		log.Warn("failed loading .env file, using environment variables", "error", err)
	}
	pwd := os.Getenv("DBPASSWORD")
	if pwd == "" {
		return nil, fmt.Errorf("DBPASSWORD enviroment variable is required")
	}

	connString := fmt.Sprintf("postgres://postgres:%s@localhost:5432/%s", pwd, dbName)

	pool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		log.Error("Failed to connect to database",
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Error("failed to ping database", "error", err)
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	s := Storage{
		db:          pool,
		log:         log,
		newsChecker: newsChecker,
	}

	log.Info("Storage initialized successfully")
	return &s, nil
}

// AddComment добавляет комментарий в БД
func (s *Storage) AddComment(ctx context.Context, newsID int, comment string) error {
	exists, err := s.newsChecker.NewsExists(ctx, newsID)
	if err != nil {
		s.log.Error("Failed to check news existence", "newsID", newsID, "error", err)
		return fmt.Errorf("failed to check news: %w", err)
	}
	if !exists {
		err := fmt.Errorf("news with ID %d does not exist", newsID)
		s.log.Error("news with ID does not exist", "newsID", newsID, "error", err)
		return err
	}
	commentID := uuid.New()

	_, err = s.db.Exec(ctx, `INSERT INTO comments (comment_id, news_id, message, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5)`, commentID, newsID, comment, time.Now(), time.Now())
	if err != nil {
		s.log.Error("Cannot add comment to database", "newsID", newsID, "error", err)
		return fmt.Errorf("failed to add comment: %w", err)
	}

	s.log.Info("comment added successfully", "newsID", newsID, "commentID", commentID)
	return nil
}

// GetComments получает список комментариев по ID новости
func (s *Storage) GetComments(ctx context.Context, newsID int) ([]models.Comment, error) {
	if newsID < 1 {
		err := fmt.Errorf("invalid news ID: %d", newsID)
		s.log.Error("Invalid news ID", "newsID", newsID, "error", err)
		return nil, err
	}
	exists, err := s.newsChecker.NewsExists(ctx, newsID)
	if err != nil {
		s.log.Error("Failed to check news existence", "newsID", newsID, "error", err)
		return []models.Comment{}, fmt.Errorf("failed to check news: %w", err)
	}
	if !exists {
		err := fmt.Errorf("news with ID %d exists", newsID)
		s.log.Error("news with ID exists", "newsID", newsID, "error", err)
		return []models.Comment{}, err
	}

	rows, err := s.db.Query(ctx,
		`SELECT *
		FROM comments
		WHERE news_id = $1
		ORDER BY created_at;`,
		newsID)
	if err != nil {
		s.log.Error("Failed to get comments from database", "newsID", newsID, "error", err)
		return []models.Comment{}, fmt.Errorf("failed to get comments: %w", err)
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		err = rows.Scan(
			&comment.CommentID,
			&comment.NewsID,
			&comment.Content,
			&comment.CreatedAt,
		)
		if err != nil {
			s.log.Error("failed to scan row", "newsID", newsID, "error", err)
			return []models.Comment{}, fmt.Errorf("failed to scan row: %w", err)
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

func (c *NewsAPIClient) NewsExists(ctx context.Context, id int) (bool, error) {
	if id <= 0 {
		return false, fmt.Errorf("invalid news ID: %d", id)
	}
	url := fmt.Sprintf("%s/news/%d", c.BaseURL, id)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		c.log.Error("failed to create request", "error", err, "news_id", id)
		return false, fmt.Errorf("failed to create request for news %d: %w", id, err)
	}

	client := http.Client{Timeout: 7 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.log.Error("failed to execute request", "error", err, "news_id", id)
		return false, fmt.Errorf("failed to check news %d existence: %w", id, err)
	}
	defer resp.Body.Close()

	exists := resp.StatusCode == http.StatusOK
	c.log.Debug("checked news existence", "news_id", id, "exists", exists, "status", resp.StatusCode)

	return exists, nil
}
