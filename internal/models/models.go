package models

import "time"

type NewsFullDetailed struct {
	NewsID      int       `json:"news_ id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Author      string    `json:"author"`
	PublishedAt time.Time `json:"published_at"`
	Source      string    `json:"source"`
	Link        string    `json:"link"`
	Tag         []string  `json:"tag"`
}

type NewsShortDetailed struct {
	NewsID      int    `json:"news_ id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Comment struct {
	CommentID int       `json:"coment_id"`
	NewsID    int       `json:"news_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
	Cens      bool      `json:"cens"`
}

type DetailedResponse struct {
	Data  interface{} `json:"data"`
	Error error       `json:"error"`
}

type FinalResponse struct {
	News     string `json:"news"`
	Comments string `json:"comments"`
}

// Request/Response структуры для Kafka
type NewsListRequest struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Filter string `json:"filter"`
}
type NewsDetailRequest struct {
	NewsID int `json:"news_id"`
}
type CommentsRequest struct {
	NewsID int `json:"news_id"`
}
type AddCommentRequest struct {
	NewsID  int    `json:"news_id"`
	Content string `json:"content"`
}
type FilterContentRequest struct {
	Content string `json:"content"`
}
type FilterDateRequest struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}
