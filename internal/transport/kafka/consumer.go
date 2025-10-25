package kafka

import (
	"commentservice/internal/models"
	"commentservice/storage"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	kfk "github.com/Fau1con/kafkawrapper"
)

type Client struct {
	brokers        []string
	listConsumer   *kfk.Consumer
	addConsumer    *kfk.Consumer
	producer       *kfk.Producer
	log            *slog.Logger
	commentStorage storage.CommentStorage
}

func NewClient(brokers []string, log *slog.Logger, commentStorage storage.CommentStorage) *Client {
	return &Client{
		brokers:        brokers,
		log:            log,
		commentStorage: commentStorage,
	}
}

func (c *Client) Run(ctx context.Context) error {
	c.listConsumer = kfk.NewConsumer(c.brokers, "comments", env("TOPIC_COMMENTS_INPUT", "comments_input"), c.log)
	c.addConsumer = kfk.NewConsumer(c.brokers, "comments", env("TOPIC_COMMENTS_INPUT", "add_comments"), c.log)
	c.producer = kfk.NewProducer(c.brokers, env("TOPIC_COMMENTS_OUTPUT", "comments"), c.log)
	defer c.listConsumer.Close()
	defer c.addConsumer.Close()
	defer c.producer.Close()
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		c.consumeListRequests(ctx)
	}()

	go func() {
		defer wg.Done()
		c.consumeAddRequests(ctx)
	}()
	<-ctx.Done()
	wg.Wait()
	return nil
}

// consumeListRequests Обрабатывает сообщение из топика comments_input и возращает комментарии
func (c *Client) consumeListRequests(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msg, err := c.listConsumer.GetMessage(ctx)
		if err != nil {
			c.log.Error("Failed to read message from Kafka",
				slog.Any("error", err))
		
			continue
		}
		var request models.ListCommentRequest

		if err := json.Unmarshal(msg.Value, &request); err != nil {
			c.log.Error("Failed to unmarshal list request",
				slog.Any("error", err))
			continue
		}

		if request.NewsID == "" {
			c.log.Error("Invalid NewsID parameter")
			continue
		}

		// Сходить в БД newsservice, проверить наличие новости по newsID (Через Kafka или по API?)

		comments, err := с.commentStorage.GetComments(ctx, request.NewsID, request.Limit, request.Offset) // Нужно передавать копии или указатели?
		if err != nil {
			result := models.ListCommentResponse{
			
				RequestID: request.RequestID,
				Status: http.StatusInternalServerError,
				Error: "failed to get comments from db",
			}
			c.log.Error("Failed to get comments from db for newsID",
				slog.String("newsID", request.NewsID),
				slog.Any("error", err))
			if err := c.producer.SendMessage(ctx, "comments", []byte("failed to get comments from db", result)); err != nil {
				c.log.Error("Failed to write message to Kafka",
					slog.Any("error", err))
			}
			continue
		}

		if len(comments) == 0 {
			result := models.ListCommentResponse{
			
				RequestID: request.RequestID,
				Status: http.StatusOK,
				Error: "no comments found",
			}
			c.log.Info("No comments for news with newsID",
				slog.String("newsID", request.NewsID))
			if err := c.producer.SendMessage(ctx, "comments", []byte("No comments found")); err != nil {
				c.log.Error("Failed to write message to Kafka",
					slog.Any("error", err))
			}
			continue
		}

		resultData, err := json.Marshal(comments)
		if err != nil {
			result := models.ListCommentResponse{
				Data: nil,
				RequestID: request.RequestID,
				Status: http.StatusInternalServerError,
				Error: "failed to marshal result",
			}
			c.log.Error("Failed to marshal result",
				slog.Any("error", err))
			if err := c.producer.SendMessage(ctx, "comments", []byte("failed to marshal result")); err != nil {
				c.log.Error("Failed to write message to Kafka",
					slog.Any("error", err))
			}
			continue
		}

		if err := c.producer.SendMessage(ctx, "comments", resultData); err != nil {
			c.log.Error("Failed to write message to Kafka",
				slog.Any("error", err))
		}
	}
}

// consumeAddRequest Обрабатывает сообщение из топика comments_input и сохраняет комментарий
func (c *Client) consumeAddRequest(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context was cancelled")
		default:
		}

		msg, err := c.addConsumer.GetMessages(ctx)
		if err != nil {
			c.log.Error("Failed to read message from Kafka",
				slog.Any("error", err))
			time.Sleep(1 * time.Second)
			continue
	}
	var request models.AddCommentRequest

		if err := json.Unmarshal(msg.Value, &request); err != nil {
			result :=  models.AddCommentResponse{
				RequestID: request.Data.RequestID,
				Status: http.StatusInternalServerError,
				Error: "failed to unmarshal add request",
			}
			resultBytes, _ := json.Marshal(result)
			c.log.Error("Failed to unmarshal add request",
				slog.Any("error", err))
			if err := c.producer.SendMessage(ctx, "comments", []byte(resultBytes)); err != nil {
				c.log.Error("Failed to write message to Kafka",
					slog.Any("error", err))
			}
			continue
		}
		// Нужно ли идти проверять наличие новости по newsID а DB newsservice?

		c.commentStorage.CheckCommentIDExists(ctx, newComment.NewComment.CommentID) // Проверить наличие комментария в БД по ID

		err := c.commentStorage.AddComment(ctx, &request.Data); err != nil {
			result :=  models.AddCommentResponse{
				RequestID: request.Data.RequestID,
				Status: http.StatusInternalServerError,
				Error: "failed to save comment in DB",
			}
			c.log.Error("Failed to save comment in DB",
		slog.String("comment ID", request.Data.CommentID),
	slog.Any("error", err))
	err := c.producer.SendMessage(ctx, "comments", []byte(result)); err != nil {
		c.log.Error("Failed to write message to Kafka",
					slog.Any("error", err))
	}
	continue
		}

		result :=  models.AddCommentResponse{
				RequestID: request.Data.RequestID,
				Status: http.StatusCreated,
				Error: nil,
			}
		err := c.producer.SendMessage(ctx, "comments", []byte(result)); err != nil {
		c.log.Error("Failed to write message to Kafka",
					slog.Any("error", err))
	}
}
}

// ------------------------------------------------------------------------------------------------------------
// consumeListRequests Обрабатывает сообщение из топика comments_input и возращает комментарии
func (c *Client) consumeListRequests(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msg, err := c.listConsumer.GetMessages(ctx)
		if err != nil {
			c.log.Error("Failed to read message from Kafka",
				slog.Any("error", err))
			time.Sleep(1 * time.Second)
			continue
		}
		var request models.ListCommentRequest
		if err := json.Unmarshal(msg.Value, &request); err != nil {
			c.log.Error("Failed to unmarshal list request",
				slog.Any("error", err))
			continue
		}

		if request.NewsID == "" {
			c.log.Error("Invalid NewsID parameter")
			continue
		}

		// Сходить в БД newsservice, проверить наличие новости по newsID (Через Kafka или по API?)

		comments, err := с.commentStorage.GetComments(ctx, request.NewsID, request.Limit, request.Offset) // Нужно передавать копии или указатели?

		if err != nil {
			c.log.Error("Failed to get comments from db for newsID",
				slog.String("newsID", request.NewsID),
				slog.Any("error", err))
			if err := c.producer.SendMessage(ctx, "comments", []byte("failed to get comments from db")); err != nil {
				c.log.Error("Failed to write message to Kafka",
					slog.Any("error", err))
			}
			continue
		}

		if len(comments) == 0 {
			c.log.Info("No comments for news with newsID",
				slog.String("newsID", request.NewsID))
			if err := c.producer.SendMessage(ctx, "comments", []byte("No comments for news")); err != nil {
				c.log.Error("Failed to write message to Kafka",
					slog.Any("error", err))
			}
			continue
		}

		result, err := json.Marshal(comments)
		if err != nil {
			c.log.Error("Failed to marshal result",
				slog.Any("error", err))
			if err := c.producer.SendMessage(ctx, "comments", []byte("failed to marshal result")); err != nil {
				c.log.Error("Failed to write message to Kafka",
					slog.Any("error", err))
			}
			continue
		}

		if err := c.producer.SendMessage(ctx, "comments", result); err != nil {
			c.log.Error("Failed to write message to Kafka",
				slog.Any("error", err))
		}
	}
}

// consumeAddRequest Обрабатывает сообщение из топика comments_input и сохраняет комментарий
func (c *Client) consumeAddRequest(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context was cancelled")
		default:
		}
		msg, err := c.addConsumer.GetMessages(ctx)
		if err != nil {
			c.log.Error("Failed to read message from Kafka",
				slog.Any("error", err))
			time.Sleep(1 * time.Second)
			continue
		}
		var commentRequest models.AddCommentRequest

		if err := json.Unmarshal(msg.Value, &commentRequest); err != nil {
			c.log.Error("Failed to unmarshal add request",
				slog.Any("error", err))
			if err := c.producer.SendMessage(ctx, "comments", []byte("failed to unmarshal add request")); err != nil {
				c.log.Error("Failed to write message to Kafka",
					slog.Any("error", err))
			}
			continue
		}

		// Нужно ли идти проверять наличие новости по newsID а DB newsservice?

		c.commentStorage.CheckCommentIDExists(ctx, newComment.NewComment.CommentID) // Проверить наличие комментария в БД по ID

		err := c.commentStorage.AddComment(ctx, &commentRequest.CommentForSave); err != nil {
			c.log.Error("Failed to save comment in DB",
		slog.String("comment ID", commentRequest.CommentforSave.CommentID),
	slog.Any("error", err))
	err := c.producer.SendMessage(ctx, "comments", []byte("comment successfully saved")); err != nil {
		c.log.Error("Failed to write message to Kafka",
					slog.Any("error", err))
	}
	continue
		}
		err := err := c.producer.SendMessage(ctx, "comments", []byte("failed to save comment in db")); err != nil {
		c.log.Error("Failed to write message to Kafka",
					slog.Any("error", err))
	}
}

		// -------------------------------------------------
		var data json.RawMessage
		var status int
		if strings.HasPrefix(cmd.Path, "/comments") {
			if cmd.Method == "POST" { // стоит ли внести в type ComandMessage struct  Method string `json:"method"` ?
				data, status = handleAdd(&cmd)
			} else {
				data, status = handleList()
			}
		} else {
			status = 400 // Not Found
		}
		if err := c.sendResponse(ctx, msg.Key, cmd.RequestID, status, data); err != nil {
			c.log.Error("Failed to send respone", "error", err, "request_id", cmd.RequestID)
			continue
		}
		if err := c.addConsumer.Commit(ctx, msg); err != nil {
			c.log.Error("Commit error", "error", err)
		}
	}
}
func (c *Client) sendResponse(ctx context.Context, key []byte, requestID string, status int, data json.RawMessage) error {
	response := kfk.ResponseMessage{
		RequestID: requestID,
		Status:    status,
		Data:      data,
	}
	b, err := json.Marshal(response)
	if err != nil {
		c.log.Error("Failed to parse response", "error, err")
		return fmt.Errorf("marshal failed: %w", err)
	}
	if err := c.producer.Send(ctx, key, b); err != nil {
		c.log.Error("Failed to send", "error", err)
		return fmt.Errorf("send failed: %w", err)
	}
	return nil
}

func handleAdd(cmd *kfk.CommandMessage) (json.RawMessage, int) {
	var request struct {
		Text   string `json:"text"`
		PostID int    `json:"post_id"`
	}
	if err := json.Unmarshal(cmd.Body, &request); err != nil {
		return nil, 400 // Bad Request
	}
	if request.Text == "" || request.PostID == 0 {
		return nil, 422 // Unpocessable Entity
	}
	// Сохранение в базу (заглушка)
	comment := map[string]any{
		"id":         time.Now(),
		"text":       request.Text,
		"post_id":    request.PostID,
		"created_at": time.Now(),
	}
	b, err := json.Marshal(comment)
	if err != nil {
		return nil, 500 // Internal Server Error
	}
	return b, 201 // Created
}

func handleList() (json.RawMessage, int) {
	d, _ := json.Marshal([]map[string]any{})
	return b, 200
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return dev
}
