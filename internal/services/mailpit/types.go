package mailpit

import "time"

type Address struct {
	Name    string `json:"Name"`
	Address string `json:"Address"`
}

type PartData struct {
	ContentType string `json:"contentType"`
	Data        []byte `json:"data"`
}

type MessageSummary struct {
	ID          string    `json:"ID"`
	MessageID   string    `json:"MessageID"`
	From        Address   `json:"From"`
	To          []Address `json:"To"`
	Subject     string    `json:"Subject"`
	Created     time.Time `json:"Created"`
	Read        bool      `json:"Read"`
	Size        int64     `json:"Size"`
	Attachments int       `json:"Attachments"`
	Snippet     string    `json:"Snippet"`
}

type ListResult struct {
	Start         int              `json:"start"`
	Total         int              `json:"total"`
	Unread        int              `json:"unread"`
	MessagesCount int              `json:"messages_count"`
	Messages      []MessageSummary `json:"messages"`
}

type Attachment struct {
	PartID      string `json:"PartID"`
	FileName    string `json:"FileName"`
	ContentType string `json:"ContentType"`
	Size        int64  `json:"Size"`
}

type Message struct {
	ID          string       `json:"ID"`
	MessageID   string       `json:"MessageID"`
	From        Address      `json:"From"`
	To          []Address    `json:"To"`
	Cc          []Address    `json:"Cc"`
	Bcc         []Address    `json:"Bcc"`
	ReplyTo     []Address    `json:"ReplyTo"`
	Subject     string       `json:"Subject"`
	Date        time.Time    `json:"Date"`
	Text        string       `json:"Text"`
	HTML        string       `json:"HTML"`
	Size        int64        `json:"Size"`
	Attachments []Attachment `json:"Attachments"`
}
