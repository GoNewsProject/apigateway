package http

import (
	"apigateway/internal/models"
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	httputils "github.com/Fau1con/renderresponse"

	kfk "github.com/Fau1con/kafkawrapper"
)

const DEFAULT_LIMIT = "10"
const PAGE = "1"

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("GoNews Server"))
}

// HandleNewsList Враппер для хендлера
func HandleNewsList(ctx context.Context, c *kfk.Consumer, p *kfk.Producer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !httputils.ValidateMethod(w, r, http.MethodGet, http.MethodOptions) {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		pageStr := r.URL.Query().Get("page")
		if pageStr == "" {
			pageStr = PAGE
		}

		limitStr := r.URL.Query().Get("n")
		if limitStr == "" {
			limitStr = DEFAULT_LIMIT
		}

		page, err := strconv.Atoi(pageStr)
		if err != nil {
			log.Println("Invalid page parameter, used default parameter")
		}

		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			log.Println("Invalid limit parameter, used default parameter")
		}

		kafkaMessage := fmt.Sprintf("/newslist/?n=%d&page=%d", limit, page)

		err = p.SendMessage(ctx, "news_input", []byte(kafkaMessage))
		if err != nil {
			httputils.RenderError(w, "Failed to write message in Kafka", http.StatusInternalServerError)
			return
		}

		msg, err := c.GetMessages(ctx)
		if err != nil {
			httputils.RenderError(w, "Failed to read message from Kafka", http.StatusInternalServerError)
			return
		}

		httputils.RenderJSON(w, msg.Value, http.StatusOK)
	}
}

// HandleFilterContent Враппер для хендлера
func HandleFilterContent(ctx context.Context, c *kfk.Consumer, p *kfk.Producer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !httputils.ValidateMethod(w, r, http.MethodGet, http.MethodOptions) {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Собираем все возможные параметры фильтрации
		query := r.URL.Query()
		category := query.Get("category") // Нужны ли fallbacks?
		author := query.Get("author")
		date := query.Get("date")
		tags := query.Get("tags") // Нужно ли заменить на tags := query[tag] для нескольких тегов в строке?
		limit := query.Get("limit")

		var queryParams []string

		if category != "" {
			queryParams = append(queryParams, "category="+category)
		}
		if author != "" {
			queryParams = append(queryParams, "author="+author)
		}
		if date != "" {
			queryParams = append(queryParams, "date="+date)
		}
		if tags != "" {
			queryParams = append(queryParams, "tags="+tags)
		}
		if limit != "" {
			queryParams = append(queryParams, "limit="+limit)
		} else {
			queryParams = append(queryParams, "limit="+DEFAULT_LIMIT)
		}

		kafkaMessage := "newslist/filtered"
		if len(queryParams) > 0 {
			kafkaMessage += "?" + strings.Join(queryParams, "&")
		}

		err := p.SendMessage(ctx, "news_input", []byte(kafkaMessage))
		if err != nil {
			httputils.RenderError(w, "Failed to write message in Kafka", http.StatusInternalServerError)
		}

		msg, err := c.GetMessages(ctx)
		if err != nil {
			httputils.RenderError(w, "Failed to read message in Kafka", http.StatusInternalServerError)
		}

		httputils.RenderJSON(w, msg.Value, http.StatusOK)
	}
}

// HandleFilterDate Враппер для хендлера
func HandleFilterDate(ctx context.Context, c *kfk.Consumer, p *kfk.Producer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !httputils.ValidateMethod(w, r, http.MethodGet, http.MethodOptions) {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		date := r.URL.Query().Get("date")
		if date == "" {
			httputils.RenderError(w, "Invalid date parameter", http.StatusBadRequest)
		}

		err := p.SendMessage(ctx, "news_input", []byte("newslist/filtered/?date="+date))
		if err != nil {
			httputils.RenderError(w, "Failed to write message in Kafka", http.StatusInternalServerError)
		}

		msg, err := c.GetMessages(ctx)
		if err != nil {
			httputils.RenderError(w, "Failed to read message in Kafka", http.StatusInternalServerError)
		}

		httputils.RenderJSON(w, msg.Value, http.StatusOK)
	}
}

// HandleNewsDetail Враппер для хендлера
func HandleNewsDetail(ctx context.Context, detailConsumer, commentConsumer *kfk.Consumer, newsProducer, commentProducer *kfk.Producer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !httputils.ValidateMethod(w, r, http.MethodGet, http.MethodOptions) {
			return
		}

		newsID := r.URL.Query().Get("id")
		if newsID == "" {
			httputils.RenderError(w, "Invalid newsID parameter", http.StatusBadRequest)
			return
		}

		chData := make(chan models.DetailedResponse, 2)

		var wg sync.WaitGroup
		wg.Add(2)

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Получение информации по новости
		go func() {
			defer wg.Done()
			if err := detailedNewsRedirectHandler(ctx, newsID, detailConsumer, newsProducer, chData); err != nil {
				select {
				case chData <- models.DetailedResponse{Error: err}:
				default:
				}
			}
		}()

		// Получение комментариев
		go func() {
			defer wg.Done()
			if err := commentsListRedirectHandler(ctx, newsID, commentConsumer, commentProducer, chData); err != nil {
				select {
				case chData <- models.DetailedResponse{Error: err}:
				default:
				}
			}
		}()
		wg.Wait()
		close(chData)

		finalResponse, err := combineResponses(chData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError) // Стоит ли использовать renderError?
			return
		}
		httputils.RenderJSON(w, finalResponse, http.StatusOK)
	}
}

func detailedNewsRedirectHandler(ctx context.Context, newsID string, detailConsumer *kfk.Consumer, p *kfk.Producer, chData chan<- models.DetailedResponse) error {
	err := p.SendMessage(ctx, "news_input", []byte("/newsdetail/"+newsID))
	if err != nil {
		return fmt.Errorf("failed to write message in Kafka: %w", err)
	}
	msg, err := detailConsumer.GetMessages(ctx)
	if err != nil {
		return fmt.Errorf("error when reading message from Kafka: %w", err)
	}
	newsData := models.DetailedResponse{Data: string(msg.Value)}
	chData <- newsData
	return nil
}

func commentsListRedirectHandler(ctx context.Context, newsID string, c *kfk.Consumer, p *kfk.Producer, chData chan<- models.DetailedResponse) error {
	err := p.SendMessage(ctx, "comments_input", []byte("/comments/"+newsID))
	if err != nil {
		return fmt.Errorf("failed to write message in Kafka: %w", err)
	}
	msg, err := c.GetMessages(ctx)
	if err != nil {
		return fmt.Errorf("failed to read message from Kafka: %w", err)
	}
	commentsData := models.DetailedResponse{Data: string(msg.Value)}
	chData <- commentsData
	return nil
}

// Функция распределения ответов от разных сервисов
func combineResponses(chData <-chan models.DetailedResponse) (models.FinalResponse, error) {
	var finalResponse models.FinalResponse
	for response := range chData {
		v, ok := response.Data.(string)
		if !ok {
			return models.FinalResponse{}, fmt.Errorf("unexpected response type not ok: %T", v)
		}
		if strings.Contains(v, "Title") {
			finalResponse.News = v
		}
		if strings.Contains(v, "comment") {
			finalResponse.Comments = v
		}
	}
	return finalResponse, nil
}

// HandleCommentsByNews Враппер для хендлера
func HandleCommentsByNews(ctx context.Context, c *kfk.Consumer, p *kfk.Producer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !httputils.ValidateMethod(w, r, http.MethodGet, http.MethodOptions) {
			return
		}
		newsID := r.URL.Query().Get("newsID")
		if newsID == "" {
			httputils.RenderError(w, "Invalid newsID parameter", http.StatusBadRequest)
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		err := p.SendMessage(ctx, "comments_input", []byte("/comments/?newsID="+newsID))
		if err != nil {
			httputils.RenderError(w, "Failed to write message in Kafka", http.StatusInternalServerError)
		}

		msg, err := c.GetMessages(ctx)
		if err != nil {
			httputils.RenderError(w, "Failed to read message in Kafka", http.StatusInternalServerError)
		}

		httputils.RenderJSON(w, msg.Value, http.StatusOK)
	}
}

// HandleAddComment Враппер для хендлера
func HandleAddComment(ctx context.Context, c *kfk.Consumer, p *kfk.Producer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !httputils.ValidateMethod(w, r, http.MethodPost, http.MethodOptions) {
			return
		}
		comment := r.URL.Query().Get("comment")
		if comment == "" {
			httputils.RenderError(w, "Invalid comment parameter", http.StatusBadRequest)
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		err := p.SendMessage(ctx, "comment_input", []byte("/add_comments/?comment="+comment))
		if err != nil {
			httputils.RenderError(w, "Failed to write message in Kafka", http.StatusInternalServerError)
		}

		msg, err := c.GetMessages(ctx)
		if err != nil {
			httputils.RenderError(w, "Failed to read message in Kafka", http.StatusInternalServerError)
		}

		httputils.RenderJSON(w, msg.Value, http.StatusCreated)
	}
}
