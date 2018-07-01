// задача - сделать простой чатик на Golang / JavaScript.
// Многопоточный, на Websocket
// Доказать, что не валенок. Как можно быстрее, послезавтра надо сдать.

// сущности, которые есть:
// анонимный пользователь
//     цвет.
// сообщение
//     текст, цвет.
// чаткомната
//     url
//     список сообщений
//     количество пользователей

// бизнес процессы:
// человек по url входит в комнату
// может там написать сообшение
// оно добавится в конец ленты и отошлётся на сервер
// если по этому url кто-то есть - он получит сообщение тут же
// если никого нет - комната удаляется вместе с историей.
// Потому что я такого ещё не видел, а telegram уже есть.

// Что происходит при работе
// Пользователь делает запрос на корневой url
// получает статическую страничку с описанием работы
// Как там написано, переходит на любой url кроме корневого
// ему отдаётся статическая страничка
//

package main

import (
	"log"
	"net/http"
	"github.com/gorilla/websocket"
)

// единственный глобальный объект - контейнер для всех комнат.
var reactor = Reactor{
	rooms: make(map[string]*Room),
}

func main() {
	http.HandleFunc("/", userCreator)
	log.Println("Listening on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Println(err)
	}
}

// Настройки WebSocket, по умолчанию, токен не проверяем.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func userCreator(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Unable to upgrade: ", err)
		return
	}
	// создаём пользовалеля вокруг соединения.
	user := User{
		roomName:                 r.URL.Path[1:], // Так как путь всегда начинается со '/'
		lastReadId:               -1,             // что бы пользователь получил всю историю, от начала диалога.
		connection:               conn,
		haveIncomingNotification: make(chan struct{}, 10),
	}
	// регистрируем пользователя в реакторе(там же создаём ему комнату, если пользователь первый) -
	// теперь другие горутины будут знать, кому отправлять сообщения.
	err = reactor.Register(&user)

	if err != nil {
		log.Println("Unable to add user: ", err)
		conn.Close()
		return
	} else {
		log.Println("Add user to room: ", r.URL.Path)
	}
	go user.SocketReadHandler()
	go user.SocketWriteHandler()
	return
}
