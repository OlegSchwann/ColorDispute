package main

import (
	"github.com/gorilla/websocket"
	"sync"
	"log"
)

type Message struct {
	Id              int    `json:"id"`
	MessageText     string `json:"message_text"`
	BackgroundColor string `json:"background_color"`
	TextColor       string `json:"text_color"`
}

type Messages struct {
	List  []*Message // список сообщений
	Mutex sync.RWMutex
}

type User struct {
	roomName   string
	lastReadId int
	connection *websocket.Conn
	// отправляющая горутина обычно заблокирована на этом канале.
	// Если там что-то есть, она идёт за обновлениями в хранилище своей комнаты,
	// вычитывает всё, что больше lastReadId.
	haveIncomingNotification chan struct{}
}

type Users struct {
	Set   map[*User]bool // множество пользователей.
	Mutex sync.RWMutex
}

type Room struct {
	Users    Users
	Messages Messages
	// Для каждой комнаты есть горутина, заблокированная на этом канале.
	// При появлении сообщения она раскладывает в haveIncomingNotification каналы каждой горутине,
	// что отправляет данные пользователю.
	haveIncomingNotification chan struct{}
}

// реактор - контейнер для комнат.
type Reactor struct {
	// Идея на будущее: так как комнаты между собой не взаимодействуют, можно их зашардировать,
	// сделать штук 12 mutex'ов и писать не в один поток.
	mutex sync.RWMutex
	// ключ - roomName
	rooms map[string]*Room
}

// Служебная горутина, существует в каждой комнате, берёт факт нового сообщения
// из Room.haveIncomingNotification и кладёт сообщения всем пользователям
// в User.haveIncomingNotification.
func (r *Room) RoomManager() {
	for range r.haveIncomingNotification {
		r.Users.Mutex.Lock() // что бы не могли изменить список пользователей, пока мы рассылаем уведомления.
		for user := range r.Users.Set {
			func() { // запись в зактытый канал означает, что горутина скоро будет удалена, не ошибка логики.
				defer func() { recover() }()
				user.haveIncomingNotification <- struct{}{}
			}()
		}
		r.Users.Mutex.Unlock()
	}
}

func (r *Reactor) createRoom(roomName string) {
	// мьютех уже захвачен Register
	newRoom := &Room{
		Users: Users{
			Set: make(map[*User]bool),
		},
		haveIncomingNotification: make(chan struct{}, 20),
	}
	r.rooms[roomName] = newRoom
	go newRoom.RoomManager()
	return
}

// регистрирует пользователя, если комнаты нет - создаёт её.
func (r *Reactor) Register(user *User) (error) {
	// проверить комнату, если нет - создать полную цепочку
	r.mutex.Lock()
	room, ok := r.rooms[user.roomName]
	if !ok {
		r.createRoom(user.roomName)
		room = r.rooms[user.roomName]
	}
	r.mutex.Unlock()

	room.Users.Mutex.Lock()
	room.Users.Set[user] = true
	room.Users.Mutex.Unlock()

	user.haveIncomingNotification <- struct{}{} // тут, когда всё создано, пользователь получит все старые сообщения.
	return nil
}

// удаляет пользователя, если он последний в комнате - удаляет её.
func (r *Reactor) UnRegister(user *User) (error) {
	r.mutex.RLock()
	room := r.rooms[user.roomName]
	r.mutex.RUnlock()

	room.Users.Mutex.Lock()
	delete(room.Users.Set, user)
	if len(room.Users.Set) == 0 {
		r.mutex.Lock()
		delete(r.rooms, user.roomName)
		r.mutex.Unlock()
		log.Print("Комната ", user.roomName, " удалена.\n")
	}
	room.Users.Mutex.Unlock()
	return nil
}

func (r *Reactor) Send(roomName string, message Message) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	room := r.rooms[roomName]
	room.Messages.Mutex.Lock()
	defer room.Messages.Mutex.Unlock()
	message.Id = len(room.Messages.List)
	room.Messages.List = append(room.Messages.List, &message)
	room.haveIncomingNotification <- struct{}{}
	return
}

func (u *User) SocketReadHandler() {
	for {
		message := Message{}
		err := u.connection.ReadJSON(&message)
		if err != nil {
			log.Println("Пришёл не правильный json: ", err)
			break
		}
		log.Printf("От пользователя в комнате %s пришло сообщение: %+v.\n", u.roomName, message)
		reactor.Send(u.roomName, message)
	}
	close(u.haveIncomingNotification)
	u.connection.Close()
	log.Print("Пользователь больше не читает из \"", u.roomName, "\".\n")
	return
}

func (u *User) SocketWriteHandler() {
	// сразу блокируемся на канале с новыми сообщениями.
	// когда комната будет полностью создана в Reactor.Register, тут появится новое уведомление
	// как приходят, лезем в массив сообщений комнаты, отсылаем всё, что строго больше u.lastReadId
	var room *Room
	for range u.haveIncomingNotification {
		if room == nil { // только для первого раза, находим комнату.
			reactor.mutex.RLock()
			room = reactor.rooms[u.roomName]
			reactor.mutex.RUnlock()
		}
		var messagesSnapshot []*Message
		room.Messages.Mutex.RLock()
		lastAvailableId := len(room.Messages.List) - 1
		if lastAvailableId > u.lastReadId {
			// как можно меньше времени блокируем общий ресурс, для этого делаем новый slice.
			messagesSnapshot = room.Messages.List[u.lastReadId+1:]
		}
		room.Messages.Mutex.RUnlock()
		u.lastReadId = lastAvailableId
		if len(messagesSnapshot) != 0 {
			for _, message := range messagesSnapshot {
				err := u.connection.WriteJSON(*message)
				if err != nil {
					log.Println("Невозможно записать: ", err)
					break
				}
			}
		}
	}
	log.Print("Пользователь больше не пишет в \"", u.roomName, "\".\n")
	return
}
