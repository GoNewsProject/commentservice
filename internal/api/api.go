package api

import (
	"commentservice/internal/models"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Api struct {
	newsDB     *postgres.Storage
	commentsDB *postgres.Storage
	r          *mux.Router
}

func New(newsDB, commentsDB *postgres.Storage, r *mux.Router) *Api {
	api := Api{
		newsDB:     newsDB,
		commentsDB: commentsDB,
		r:          mux.NewRouter(),
	}
	api.endpoints()
	return &api
}

func (api *Api) Router() *mux.Router {
	return api.r
}

// Регистрация маршрутов
func (api *Api) endpoints() {
	//Маршрут предоставления списка комментариев по ID новости
	api.r.HandleFunc("/comments/{id}", api.GetCommentsHandler).Methods(http.MethodGet, http.MethodOptions)
	//Маршрут добавления комментария
	api.r.HandleFunc("/addComment/", api.AddCommentHandler).Methods(http.MethodPost, http.MethodOptions)
}

// GetCommentsHandler предоставляет список комментариев
func (api *Api) GetCommentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodOptions {
		return
	}
	// id == news_id
	s := mux.Vars(r)["id"]
	id, _ := strconv.Atoi(s)

	comments, err := api.commentsDB.GetComments(id, api.newsDB)
	if err != nil {
		http.Error(w, "failed get comments fromDB", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(comments)
	w.WriteHeader(http.StatusOK)
}

// AddCommentHandler добавляет комментарий
func (api *Api) AddCommentHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", http.MethodPost)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", http.MethodPost)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}
	// создание объекта комментария из тела запроса
	var comment models.Comment
	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		http.Error(w, "failed to decode JSON payload", http.StatusBadRequest)
	}
	err := api.commentsDB.AddComment(comment, api.newsDB)
	if err != nil {
		log.Printf("Cant add comment to database: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("the comment has been added successfully!"))
}
