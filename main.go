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
	"fmt"
)

func main() {
	http.HandleFunc("/", socketCreator)
	log.Println("Listening on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Println(err)
	}
}

// Настройки буферов WebSocket.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func socketCreator(w http.ResponseWriter, r *http.Request) {
	roomName := r.URL.Path
	println(roomName)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Unable to upgrade: ", err)
		return
	}
	// TODO: регистрация в реакторе по roomName
	go socketReadHandler(conn)
	go socketWriteHandler(conn)
	return
}

func socketReadHandler(conn *websocket.Conn)  {
	for {
		message := Message{}
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Println("Пришёл не правильный json: ", err)
			break
		}
		fmt.Printf("%v", message)
	}
	conn.Close()
	return
}

func socketWriteHandler(conn *websocket.Conn)  {
	err := conn.WriteJSON(Message{
		Id:              1,
		MessageText:     "I just came to say hello!",
		BackgroundColor: "#ddfeff",
		TextColor:       "#14243d",
	})
	if err != nil {
		log.Println("Unable to write:", err)
	}
	//conn.Close()
	return
}
