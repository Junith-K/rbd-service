package models

import "time"

// History represents a notification trigger event
type History struct {
	HistoryID      string    `firestore:"historyId" json:"historyId"`
	SenderID       string    `firestore:"senderId" json:"senderId"`
	ReceiverID     string    `firestore:"receiverId" json:"receiverId"`
	SenderUsername string    `firestore:"senderUsername" json:"senderUsername"`
	TriggeredAt    time.Time `firestore:"triggeredAt" json:"triggeredAt"`
}

// HistoryResponse represents the history listing response
type HistoryResponse struct {
	History []History `json:"history"`
	Total   int       `json:"total"`
}
