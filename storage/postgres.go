package storage

import (
	"commentservice/internal/models"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
)

type Storage struct {
	DB  *pgxpool.Pool
	log *slog.Logger
}

// NewsAPIClient для проверки новостей через API
type NewsAPIClient struct {
	BaseURL string
}

func NewNewsAPIClient(baseURL string) *NewsAPIClient {
	return &NewsAPIClient{BaseURL: baseURL}
}

func (c *NewsAPIClient) NewsExists(ctx context.Context, id int) (bool, error) {
	url := fmt.Sprintf("%s/news/%d", c.BaseURL, id)

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func NewDB(ctx context.Context, DBname string, log *slog.Logger) (*Storage, error) {
	err := godotenv.Load()
	if err != nil {
		log.Error("Failed loading .env file", err)
		return nil, err
	}
	pwd := os.Getenv("DBPASSWORD")
	connString := "postgres://postgres:" + pwd + "@localhost:5432/" + "DBName"

	pool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		log.Error("Failed to connect to BD", err)
		return nil, err
	}
	s := Storage{
		DB:  pool,
		log: log,
	}
	return &s, nil
}

// newsIdCheck проверяет наличия новости по ID
func newsIDCheck(ctx context.Context, newsStorage *Storage, id int, log *slog.Logger) bool {
	var exists bool = true
	err := newsStorage.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM news WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		log.Error("Failed scaning row", err)
		return false
	}
	if !exists {
		log.Warn("news with id ID %d does not exist", id)
		return false
	}
	return true
}

// commentIDCheck проверяет наличие комментария по ID комментария
func commentIDCheck(id int, commentDB *Storage, log *slog.Logger) bool {
	var exists bool = true
	row := commentDB.DB.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM comments WHERE id = $1)`, id)
	err := row.Scan(&exists)
	if id == 0 {
		return true
	}
	if err != nil {
		log.Error("Failed scaning row")
		return false
	}
	if !exists {
		log.Warn("comments with ID %d does not exist", id)
		return false
	}
	return true
}

// AddComment добавляет комментарий в БД
func (s *Storage) AddComment(ctx context.Context, newsChecker NewsChecker, postID int, comment *models.Comment, log *slog.Logger) error {
	exists, err := newsChecker.NewsExists(ctx, postID)
	if err != nil {
		log.Error("Failed to check news existence", "postID", postID, "error", err)
		return fmt.Errorf("failed to check news: %w", err)
	}
	if !exists {
		err := fmt.Errorf("post with ID %d not exist", postID)
		log.Error("Post with ID does not exist", "postID", postID, "error", err)
		return err
	}

	unixTime := time.Now().Unix()
	_, err = s.DB.Exec(ctx, `INSERT INTO comments (post_id, message, created_at, updated_at)
	VALUES ($1, $2, $3, $4)`, postID, comment.Message, unixTime, unixTime)
	if err != nil {
		log.Error("Cant add comment in database", "error", err)
		return err
	}
	return nil
}

// GetComments получает список комментариев по ID новости
func (s *Storage) GetComments(ctx context.Context, newsID int, newsStore *Storage, log *slog.Logger) ([]*models.Comment, err error) {
	if !newsIDCheck(ctx, newsID, newsStore, log) {
		err := errors.New("The post with ID does not exist")
		log.Error("The post with ID %d does not exist", newsID, err)
		return []*models.Comment{}, err
	}
	if newsID < 1 {
		err := fmt.Errorf("bad news ID: %v", newsID)
		log.Error("error", err)
		return []*models.Comment{}, err
	}
	rows, err := s.DB.Query(context.Background(), `SELECT * FROM comments WHERE news_id = $1
	ORDER BY created_at`, newsID)
	if err != nil {
		log.Error("Failed to get comments from database", err)
		return []*models.Comment{}, err
	}
	defer rows.Close()

	for rows.Next() {
		comment := &models.Comment{}
		err = rows.Scan(
			&comment.CommentID,
			&comment.PostID,
			&comment.Message,
			&comment.CreatedAt,
			&comment.Parrent_id,
		)
		if err != nil {
			log.Error("Failed to scan row", err)
		}
		comments = append(comments, comment)
	}
	return comments, nil
}
