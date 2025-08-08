package chat

type Message struct {
	ChannelID string `json:"channel_id"`
	SenderID  string `json:"sender_id"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

