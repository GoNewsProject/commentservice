package storage

import (
	"commentservice/internal/infrastructure/config"
	"commentservice/internal/models"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

// newStorage внутренняя функция создания хранилища
func newStorage(dbConfig config.DBConfig, dbName string, log *slog.Logger) (*Storage, error) {
	connStr := dbConfig.GetDSN()
	log.Debug("Connecting to database",
		"database", dbName,
		"dsn", fmt.Sprintf("postgres://%s:***@%s:%d/%s",
			dbConfig.UserName, dbConfig.Host, dbConfig.Port, dbConfig.DBName))

	db, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection to database %s: %w", dbName, err)
	}

	if err := db.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping %s database: %w", dbName, err)
	}

	log.Info("database connection established",
		"database", dbName,
		"host", dbConfig.Host,
		"port", dbConfig.Port,
		"database", dbConfig.DBName)

	return &Storage{
		db:  db,
		log: log,
	}, nil
}

// NewCommentsStorage создает подключение к базе комментариев
func NewCommentStorage(cfg *config.Config, log *slog.Logger) (*Storage, error) {
	dbConfig := cfg.GetCommentsDBConfig()
	return newStorage(dbConfig, "comments", log)
}

// NewNewsStorage создает подключение к базе новостей
func NewNewsStorage(cfg *config.Config, log *slog.Logger) (*Storage, error) {
	dbConfig := cfg.GetNewsDBConfig()
	return newStorage(dbConfig, "news", log)
}

// type StorageType string

// const (
// 	StorageTypeComments StorageType = "comments"
// 	StorageTypeNews     StorageType = "news"
// )

// type Storage struct {
// 	db          *pgxpool.Pool
// 	log         *slog.Logger
// 	storageType StorageType
// 	config      config.DBConfig
// }

// // NewStorage создает подключение к указанной базе данных
// func NewStorage(storageType StorageType, cfg *config.Config, log *slog.Logger) (*Storage, error) {
// 	var dbConfig config.DBConfig
// 	var dbName string

// 	switch storageType {
// 	case StorageTypeComments:
// 		dbConfig = cfg.GetCommentsDBConfig()
// 		dbName = "comments"
// 	case StorageTypeNews:
// 		dbConfig = cfg.GetNewsDBConfig()
// 		dbName = "news"
// 	default:
// 		return nil, fmt.Errorf("unknow storage type: %s", storageType)
// 	}

// 	connStr := dbConfig.GetDSN()
// 	log.Debug("connecting to database",
// 		"type", storageType,
// 		"dsn", fmt.Sprintf("postgres://%s:***@%s:%d/%s",
// 			dbConfig.UserName, dbConfig.Host, dbConfig.Port, dbConfig.DBName))

// 	db, err := pgxpool.New(context.Background(), connStr)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create connection pool for %s: %w", dbName, err)
// 	}

// 	if err := db.Ping(context.Background()); err != nil {
// 		return nil, fmt.Errorf("failed to ping %s database: %w", dbName, err)
// 	}

// 	log.Info("database connection established",
// 		"type", storageType,
// 		"host", dbConfig.Host,
// 		"port", dbConfig.Port,
// 		"database", dbConfig.DBName)

// 	return &Storage{
// 		db:          db,
// 		log:         log,
// 		storageType: storageType,
// 		config:      dbConfig,
// 	}, nil
// }

// AddComment добавляет комментарий в БД
func (s *Storage) AddComment(ctx context.Context, newsID int, comment string) error {
	_, err := s.db.Exec(ctx, `INSERT INTO comments (news_id, message, created_at, updated_at)
	VALUES ($1, $2, $3, $4)`, newsID, comment, time.Now(), time.Now())
	if err != nil {
		s.log.Error("failed to save comment to database", "newsID", newsID, "error", err)
		return fmt.Errorf("failed to save comment: %w", err)
	}

	s.log.Info("comment added successfully", "newsID", newsID)
	return nil
}

// GetComments получает список комментариев по ID новости
func (s *Storage) GetComments(ctx context.Context, newsID int) ([]models.Comment, error) {
	if newsID < 1 {
		err := fmt.Errorf("invalid news ID: %d", newsID)
		s.log.Error("Invalid news ID", "newsID", newsID, "error", err)
		return nil, err
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, news_id, content, created_at
		FROM comments
		WHERE news_id = $1
		ORDER BY created_at;`,
		newsID)
	if err != nil {
		s.log.Error("failed to get comments from database", "newsID", newsID, "error", err)
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

func (s *Storage) NewsExists(ctx context.Context, id int) (bool, error) {
	if id <= 0 {
		return false, fmt.Errorf("invalid news ID: %d", id)
	}

	var exists bool

	query := `SELECT EXISTS(SELECT 1 FROM news WHERE id = $1)`

	err := s.db.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		s.log.Error("failed to check news existence in database", "news_id", id, "error", err)
		return false, fmt.Errorf("failed to check news existence: %w", err)
	}

	return exists, nil
}

func (s *Storage) Close() {
	if s.db != nil {
		s.db.Close()
		s.log.Info("database connection closed",
			"database", "comments/news")
	}
}
