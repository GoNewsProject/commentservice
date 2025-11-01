package http

import (
	"commentservice/storage"
	"context"
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
		params, err := parseURLParams(string(msg.Value))
		if err != nil {
			h.producer.SendMessage(ctx, "comments", []byte("failed to parse URL params"))
			httputils.RenderError(w, "failed to parse URL params", http.StatusInternalServerError)
			return
		}

		newsIDStr, exists := params["newsID"]
		if !exists || newsIDStr == "" {
			h.producer.SendMessage(ctx, "comments", []byte("news ID parameter is required"))
			httputils.RenderError(w, "news ID parameter is required", http.StatusBadRequest)
			return
		}
		newsID, err := strconv.Atoi(newsIDStr)
		if err != nil {
			h.producer.SendMessage(ctx, "comments", []byte("failed to parse news ID parameter"))
			httputils.RenderError(w, "failed to parse news ID parameter", http.StatusBadRequest)
			return
		}

		comments, err := h.storage.GetComments(ctx, newsID)
		if err != nil {
			h.producer.SendMessage(ctx, "comments", []byte("failed to get comments from database"))
			httputils.RenderError(w, "failed to get comments from database", http.StatusInternalServerError)
			return
		}

		h.producer.SendMessage(ctx, "comments", []byte(fmt.Sprintf("Retrived comments for news ID %d", newsID)))
		httputils.RenderJSON(w, comments, http.StatusOK)
	}
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

func parseURLParams(input string) (map[string]string, error) {
	parts := strings.Split(input, "?")
	if len(parts) < 2 {
		return nil, fmt.Errorf("no query parameters found")
	}

	values, err := url.ParseQuery(parts[1])
	if err != nil {
		return nil, err
	}

	params := make(map[string]string)
	for key, value := range values {
		if len(value) > 0 {
			params[key] = value[0]
		}
	}

	return params, nil
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
