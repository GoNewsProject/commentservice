package http

import (
	"context"
	"net/http"
	"time"

	httputils "github.com/Fau1con/renderresponse"

	kfk "github.com/Fau1con/kafkawrapper"
)

func HandleCommentsList(ctx context.Context, c *kfk.Consumer, p *kfk.Producer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !httputils.ValidateMethod(w, r, http.MethodGet, http.MethodOptions) {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// newsID := r.URL.Query().Get("newsID")
		// if newsID == "" {
		// 	httputils.RenderError(w, "Invalid newsId parameter", http.StatusBadRequest)
		// 	return
		// }

		msg, err := c.GetMessages(ctx)
		if err != nil {
			httputils.RenderError(w, "Failed to read message from Kafka", http.StatusInternalServerError)
		}
	}
}
