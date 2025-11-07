package api

import (
	"commentservice/internal/service"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	httputils "github.com/Fau1con/renderresponse"
	"github.com/gorilla/mux"
)

type Api struct {
	r              *mux.Router
	commentService service.CommentService
}

func NewApi(r *mux.Router, commentService service.CommentService) *Api {
	api := Api{
		r:              mux.NewRouter(),
		commentService: commentService,
	}
	api.endpoints()
	return &api
}

func (api *Api) Router() http.Handler {
	return api.r
}

// Метод регистратор endpoint-ов
func (api *Api) endpoints() {
	// маршрут предоставления списка комментариев по newsID
	api.r.HandleFunc("/comments/?newsID=", api.getComments)
	// маршрут добавления комментария
	api.r.HandleFunc("/addComment/?newsID=&comment=", api.addComment)
}

func (api *Api) getComments(w http.ResponseWriter, r *http.Request) {
	if !httputils.ValidateMethod(w, r, http.MethodGet, http.MethodOptions) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	params, err := parseURLParams(r.URL.String())
	if err != nil {
		httputils.RenderError(w, "failed to parse query parameters", http.StatusBadRequest, err)
		return
	}
	newsIDStr, exists := params["newsID"]
	if !exists {
		httputils.RenderError(w, "newsID parameter not found", http.StatusBadRequest)
		return
	}
	newsID, err := strconv.Atoi(newsIDStr)
	if err != nil {
		httputils.RenderError(w, "failed to parse newsID", http.StatusBadRequest, err)
		return
	}

	comments, err := api.commentService.GetComments(ctx, newsID)
	if err != nil {
		httputils.RenderError(w, "failed to get comments from database", http.StatusInternalServerError, err)
		return
	}

	httputils.RenderJSON(w, comments, http.StatusOK)
}

func (api *Api) addComment(w http.ResponseWriter, r *http.Request) {
	if !httputils.ValidateMethod(w, r, http.MethodPost, http.MethodOptions) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	params, err := parseURLParams(r.URL.String())
	if err != nil {
		httputils.RenderError(w, "failed to parse query parameters from URL", http.StatusBadRequest, err)
		return
	}

	newsIDStr, exists := params["newsID"]
	if !exists {
		httputils.RenderError(w, "newsID parameter not found", http.StatusBadRequest)
		return
	}
	newsID, err := strconv.Atoi(newsIDStr)
	if err != nil {
		httputils.RenderError(w, "failed to parse newsID", http.StatusBadRequest, err)
		return
	}

	comment, exists := params["comment"]
	if !exists {
		httputils.RenderError(w, "comment not found", http.StatusBadRequest)
	}
	if comment == "" {
		httputils.RenderError(w, "invalid comment", http.StatusBadRequest)
		return
	}

	err = api.commentService.AddComment(ctx, newsID, comment)
	if err != nil {
		httputils.RenderError(w, "failed to save comment to database", http.StatusInternalServerError, err)
		return
	}

	httputils.RenderJSON(w, "comment saved successfully", http.StatusCreated)

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

// type Api struct {
// 	newsDB             *storage.NewsAPIClient
// 	commentsDB         storage.CommentStorage
// 	r                  *mux.Router
// 	ctx                context.Context
// 	log                *slog.Logger
// 	commentHandler     *transport.CommentHandler
// 	commentProducer    *kfk.Producer
// 	commentConsumer    *kfk.Consumer
// 	addCommentConsumer *kfk.Consumer
// }

// type Topics struct {
// 	CommentInput string
// 	AddComment   string
// 	Comments     string
// }

// func New(
// 	newsDB *storage.NewsAPIClient,
// 	commentsDB storage.CommentStorage,
// 	r *mux.Router,
// 	ctx context.Context,
// 	log *slog.Logger,
// 	commentProducer *kfk.Producer,
// 	commentConsumer *kfk.Consumer,
// 	addCommentConsumer *kfk.Consumer,
// ) *Api {
// 	api := &Api{
// 		newsDB:             newsDB,
// 		commentsDB:         commentsDB,
// 		r:                  mux.NewRouter(),
// 		ctx:                ctx,
// 		log:                log,
// 		commentProducer:    commentProducer,
// 		commentConsumer:    commentConsumer,
// 		addCommentConsumer: addCommentConsumer,
// 	}
// 	api.commentHandler = transport.NewCommentHandler(commentsDB, commentProducer)
// 	api.endpoints()
// 	return api
// }

// func (api *Api) Router() *mux.Router {
// 	return api.r
// }

// // Регистрация маршрутов
// func (api *Api) endpoints() {
// 	//Маршрут предоставления списка комментариев по ID новости
// 	api.r.HandleFunc("/comments/?newsID=", api.commentHandler.HandleCommentsGet(api.ctx, api.commentConsumer)).Methods(http.MethodGet, http.MethodOptions)
// 	//Маршрут добавления комментария
// 	api.r.HandleFunc("/addComment/", api.commentHandler.HandleFuncCommentAdd(api.ctx, api.addCommentConsumer)).Methods(http.MethodPost, http.MethodOptions)
// }
