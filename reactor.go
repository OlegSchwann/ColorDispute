package main

import (
	"github.com/gorilla/websocket"
	"sync"
)

type Message struct {
	Id              int    `json:"id"`
	MessageText     string `json:"message_text"`
	BackgroundColor string `json:"background_color"`
	TextColor       string `json:"text_color"`
}

type Messages struct {
	List  []Message // список сообщений
	Mutex sync.RWMutex
}

type User struct {
	roomName             string
	lastReadedId         int
	connection           *websocket.Conn
	IncomingNotification *chan struct{}
}

type Users struct {
	List  []User
	Mutex sync.RWMutex
}

type Room struct {
	Users    Users
	Messages Messages
}

// реактор - система, хранящая в себе соответствия имён комнат, соединений-пользователей, сообщений
type Reactor struct {
	// ключ - roomName
	// значение - Room
	Rooms sync.Map
}


