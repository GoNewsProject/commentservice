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
	newsDB         *storage.NewsAPIClient
	commentsDB     storage.CommentStorage
	r              *mux.Router
	ctx            context.Context
	log            *slog.Logger
	commentHandler *transport.CommentHandler
	consumer       *kfk.Consumer
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
	consumer *kfk.Consumer,
	producer *kfk.Producer,
) *Api {
	api := Api{
		newsDB:     newsDB,
		commentsDB: commentsDB,
		r:          mux.NewRouter(),
		ctx:        ctx,
		log:        log,
		consumer:   consumer,
	}
	api.commentHandler = transport.NewCommentHandler(commentsDB, producer)
	api.endpoints()
	return &api
}

func (api *Api) Router() *mux.Router {
	return api.r
}

// Регистрация маршрутов
func (api *Api) endpoints() {
	//Маршрут предоставления списка комментариев по ID новости
	api.r.HandleFunc("/comments/?newsID=", api.commentHandler.HandleCommentsGet(api.ctx, api.consumer)).Methods(http.MethodGet, http.MethodOptions)
	//Маршрут добавления комментария
	api.r.HandleFunc("/addComment/", api.commentHandler.HandleFuncCommentAdd(api.ctx, api.consumer)).Methods(http.MethodPost, http.MethodOptions)
}
