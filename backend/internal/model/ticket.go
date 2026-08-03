package model

import "time"

type SupportTicket struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	SaleID    *string   `json:"sale_id,omitempty"`
	Subject   string    `json:"subject"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type TicketMessage struct {
	ID        string    `json:"id"`
	TicketID  string    `json:"ticket_id"`
	AuthorID  string    `json:"author_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTicketInput struct {
	SaleID  *string `json:"sale_id,omitempty"`
	Subject string  `json:"subject"`
	Message string  `json:"message"`
}

type CreateTicketMessageInput struct {
	Body string `json:"body"`
}
