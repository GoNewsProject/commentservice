package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	kfk "github.com/Fau1con/kafkawrapper"
)

type Client struct {
	brokers      []string
	listConsumer *kfk.Consumer
	addConsumer  *kfk.Consumer
	producer     *kfk.Producer
	log          *slog.Logger
}

func NewClient(brokers []string, log *slog.Logger) *Client {
	return &Client{
		brokers: brokers,
		log:     log,
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

func (c *Client) consumeListRequests(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
		default:
		}
		msg, err := c.listConsumer.Fetch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.log.Error("AddConsumer fetch error", "error", err)
				return
			}
			time.Sleep(1 * time.Second)
			continue
		}
		var cmd kfk.CommandMessage
		if err := json.Unmarshal(msg.Value, &cmd); err != nil { //где поле msg.Value
			_ = c.listConsumer.Commit(ctx, msg)
			continue
		}
		var data json.RawMessage
		var status int
		if strings.HasPrefix(cmd.Path, "/comments") {
			data, status = handleList()
		} else {
			status = 400 // Bad Request
		}
		if err := c.sendResponse(ctx, msg.Key, cmd.RequestID, status, data); err != nil {
			c.log.Error("Failed to send response", "error", err, "request_id", cmd.RequestID)
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

func (c *Client) consumeAddRequest(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
		default:
		}
		msg, err := c.addConsumer.Fetch(ctx)
		if err != nil {
			if ctx.Err() != nil { // можно ли использовать if errors.Is(ctx.Err(), context.Canceled) ?
				c.log.Error("AddConsumer Fetch error", "error", err)
				return fmt.Errorf("fetch error: %w", err)
			}
			time.Sleep(1 * time.Second)
			continue
		}
		var cmd kfk.CommandMessage
		if err := json.Unmarshal(msg.Value, &cmd); err != nil {
			_ = c.addConsumer.Commit(ctx, msg)
			continue
		}
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

func handleAdd(cmd *kfk.CommandMessage) (json.RawMessage, int) {
	var request struct {
		Text string `json:"text"`
		UserID int `json:"user_id"`
		PostID int `json:"post_id"`
	}
	if err := json.Unmarshal(cmd.Body, &request); err != nil {
	return nil, 400 // Bad Request
	}
	if request.Text == "" || request.UserID == 0 || request.PostID == 0 {
		return nil, 422 // Unpocessable Entity
	}
	// Сохранение в базу (заглушка)
	comment := map[string]any{
		"id": time.Now(),
		"text": request.Text,
		"user_id": request.UserID.
		"post_id": request.PostID,
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