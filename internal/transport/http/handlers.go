package http

import (
	"commentservice/internal/models"
	"commentservice/storage"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	kfk "github.com/Fau1con/kafkawrapper"
	httputils "github.com/Fau1con/renderresponse"
)

type CommentHandler struct {
	storage  storage.CommentStorage
	producer *kfk.Producer
}

func NewCommentHandler(storage storage.CommentStorage, producer *kfk.Producer) *CommentHandler {
	return &CommentHandler{
		storage:  storage,
		producer: producer,
	}
}

// HandleCommentsGet предоставляет список комментариев
func (h *CommentHandler) HandleCommentsGet(ctx context.Context, c *kfk.Consumer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !httputils.ValidateMethod(w, r, http.MethodGet) {
			h.producer.SendMessage(ctx, "comments", []byte("failed to check request method"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		msg, err := c.GetMessages(ctx)
		if err != nil {
			h.producer.SendMessage(ctx, "comments", []byte("failed to read message from Kafka"))
			httputils.RenderError(w, "failed to read message from Kafka", http.StatusInternalServerError)
			return
		}
		result := strings.TrimPrefix(string(msg.Value), "/comments/")
		newsID, err := strconv.Atoi(result)
		if err != nil {
			h.producer.SendMessage(ctx, "comments", []byte("failed to parse newsID"))
			httputils.RenderError(w, "failed to parse newsID", http.StatusBadRequest)
			return
		}
		comments, err := h.storage.GetComments(ctx, newsID)
		if err != nil {
			h.producer.SendMessage(ctx, "comments", []byte("failed to get comments from database"))
			httputils.RenderError(w, "failed to get comments from database", http.StatusInternalServerError)
			return
		}

		respResult, err := commentsToBytes(comments)
		if err != nil {
			h.producer.SendMessage(ctx, "comments", []byte("failed to marshal response"))
			httputils.RenderJSON(w, "failed to marshal response", http.StatusInternalServerError)
			return
		}

		h.producer.SendMessage(ctx, "comments", []byte(respResult))
		httputils.RenderJSON(w, comments, http.StatusOK)
	}
}

func commentsToBytes(comments []models.Comment) ([]byte, error) {
	result, err := json.Marshal(comments)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// HandleFuncCommentAdd добавляет комментарий
func (h *CommentHandler) HandleFuncCommentAdd(ctx context.Context, c *kfk.Consumer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !httputils.ValidateMethod(w, r, http.MethodPost) {
			h.producer.SendMessage(ctx, "add_comment", []byte("failed to check request method"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		msg, err := c.GetMessages(ctx)
		if err != nil {
			h.producer.SendMessage(ctx, "add_comment", []byte("failed to read message from Kafka"))
			httputils.RenderError(w, "failed to read message from Kafka", http.StatusInternalServerError)
			return
		}
		comment, newsIDStr, err := parseKafkaMessage(msg.Value)
		if err != nil {
			h.producer.SendMessage(ctx, "add_comment", []byte("failed to parse query params"))
			httputils.RenderError(w, "failed to parse query params", http.StatusInternalServerError)
			return
		}

		newsID, err := strconv.Atoi(newsIDStr)
		if err != nil {
			h.producer.SendMessage(ctx, "add_comment", []byte("failed to parse newsID"))
			httputils.RenderError(w, "failed to parse newsID", http.StatusBadRequest)
			return
		}

		err = h.storage.AddComment(ctx, newsID, comment)
		if err != nil {
			h.producer.SendMessage(ctx, "add_comment", []byte("failed to save comment in database"))
			httputils.RenderError(w, "failed to save comment in database", http.StatusInternalServerError)
			return
		}

		h.producer.SendMessage(ctx, "add_comment", []byte("comment has been added to database successfully"))
		httputils.RenderJSON(w, "comment has been added to database successfully", http.StatusCreated)
	}
}

func parseKafkaMessage(msgValue []byte) (string, string, error) {
	urlStr := string(msgValue)

	queryStart := strings.Index(urlStr, "?")
	if queryStart == -1 {
		return "", "", fmt.Errorf("no query parameters found")
	}

	queryStr := urlStr[queryStart+1:]
	value, err := url.ParseQuery(queryStr)
	if err != nil {
		return "", "", fmt.Errorf("error parsing query: %v", err)
	}

	comment := value.Get("comment")
	newsID := value.Get("news_id")

	if comment == "" || newsID == "" {
		return "", "", fmt.Errorf("parameters is required")
	}

	return comment, newsID, nil
}
