package api

import (
	"apigateway/internal/models"
	transport "apigateway/internal/transport/http"
	"context"
	"log/slog"
	"net/http"

	kfk "github.com/Fau1con/kafkawrapper"
)

type Api struct {
	mux              *http.ServeMux
	newsProducer     *kfk.Producer
	commentProducer  *kfk.Producer
	detailConsumer   *kfk.Consumer
	listConsumer     *kfk.Consumer
	commentsConsumer *kfk.Consumer
	filteredContent  *kfk.Consumer
	filterPublished  *kfk.Consumer
	responseChan     chan models.DetailedResponse
	defaultLimit     int
	ctx              context.Context
	log              *slog.Logger
	topics           Topics
}

type Topics struct {
	NewsInput     string
	CommentsInput string
	AddComments   string
}

func New(
	ctx context.Context, resp chan models.DetailedResponse,
	newsProducer, commentProducer *kfk.Producer,
	detailConsumer, listConsumer, commentsConsumer, filteredContent, filterPublished *kfk.Consumer,
	log *slog.Logger, topics Topics, limit int,
) *Api {
	api := &Api{
		mux:              http.NewServeMux(),
		newsProducer:     newsProducer,
		commentProducer:  commentProducer,
		detailConsumer:   detailConsumer,
		listConsumer:     listConsumer,
		commentsConsumer: commentsConsumer,
		filteredContent:  filteredContent,
		filterPublished:  filterPublished,
		responseChan:     resp,
		ctx:              ctx,
		log:              log,
		topics:           topics,
		defaultLimit:     limit,
	}
	api.registerRoutes()
	return api
}

// registerRoutes навешивает HTTP-маршруты.
func (a *Api) registerRoutes() {
	a.mux.HandleFunc("/", transport.HandleRoot)
	a.mux.HandleFunc("/newslist/", transport.HandleNewsList(a.ctx, a.listConsumer, a.newsProducer))
	a.mux.HandleFunc("/newslist/filtered/", transport.HandleFilterContent(a.ctx, a.listConsumer, a.newsProducer))
	a.mux.HandleFunc("/newslist/filtered/date", transport.HandleFilterDate(a.ctx, a.listConsumer, a.newsProducer))
	a.mux.HandleFunc("/newsdetail", transport.HandleNewsDetail(a.ctx, a.listConsumer, a.commentsConsumer, a.newsProducer, a.commentProducer))
	a.mux.HandleFunc("/comments/", transport.HandleCommentsByNews(a.ctx, a.commentsConsumer, a.commentProducer))
	a.mux.HandleFunc("/addcomment/", transport.HandleAddComment(a.ctx, a.commentsConsumer, a.commentProducer))
}

func (a *Api) Router() http.Handler {
	return a.mux
}

func (a *Api) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}
