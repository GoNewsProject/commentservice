package models

import "time"

type Comment struct {
	CommentID int       `json:"coment_id"`
	NewsID    int       `json:"news_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Cens      bool      `json:"cens"`
}

// Request/Response структуры для Kafka
type ListCommentRequest struct {
	NewsID    string `json:"news_id"`
	Limit     int    `json:"limit"`
	Offset    int    `json:"offset"`
	RequestID string `json:"request_id"`
}

type ListCommentResponse struct {
	Data      []Comment `json:"data"`
	RequestID string    `json:"request_id"`
	Status    string    `json:"status"`
	Error     string    `json:"error"`
}

type AddCommentRequest struct {
	Data      Comment `json:"new_comment"`
	RequestID string  `json:"request_id"`
}

type AddCommentResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Error     string `json:"error"`
}
