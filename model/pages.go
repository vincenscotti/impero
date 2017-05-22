package model

type HeaderData struct {
	CurrentPlayer *Player
	NewMessages   int
}

type GameHomeData struct {
	*HeaderData
}

type MessagesData struct {
	*HeaderData
	Messages []*Message
}

type MessageData struct {
	*HeaderData
	Message *Message
}
