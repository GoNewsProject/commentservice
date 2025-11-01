package api

import (
	transport "commentservice/internal/transport/http"
	"commentservice/storage"
	"context"
	"log/slog"
	"net/http"

	kfk "github.com/Fau1con/kafkawrapper"
	"github.com/gorilla/mux"
)

type Api struct {
	newsDB             *storage.NewsAPIClient
	commentsDB         storage.CommentStorage
	r                  *mux.Router
	ctx                context.Context
	log                *slog.Logger
	commentHandler     *transport.CommentHandler
	commentProducer    *kfk.Producer
	commentConsumer    *kfk.Consumer
	addCommentConsumer *kfk.Consumer
}

type Topics struct {
	CommentInput string
	AddComment   string
	Comments     string
}

func New(
	newsDB *storage.NewsAPIClient,
	commentsDB storage.CommentStorage,
	r *mux.Router,
	ctx context.Context,
	log *slog.Logger,
	commentProducer *kfk.Producer,
	commentConsumer *kfk.Consumer,
	addCommentConsumer *kfk.Consumer,
) *Api {
	api := &Api{
		newsDB:             newsDB,
		commentsDB:         commentsDB,
		r:                  mux.NewRouter(),
		ctx:                ctx,
		log:                log,
		commentProducer:    commentProducer,
		commentConsumer:    commentConsumer,
		addCommentConsumer: addCommentConsumer,
	}
	api.commentHandler = transport.NewCommentHandler(commentsDB, commentProducer)
	api.endpoints()
	return api
}

func (api *Api) Router() *mux.Router {
	return api.r
}

// Регистрация маршрутов
func (api *Api) endpoints() {
	//Маршрут предоставления списка комментариев по ID новости
	api.r.HandleFunc("/comments/?newsID=", api.commentHandler.HandleCommentsGet(api.ctx, api.commentConsumer)).Methods(http.MethodGet, http.MethodOptions)
	//Маршрут добавления комментария
	api.r.HandleFunc("/addComment/", api.commentHandler.HandleFuncCommentAdd(api.ctx, api.addCommentConsumer)).Methods(http.MethodPost, http.MethodOptions)
}
