package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"errors"
)

const (
	system = "system"
	all    = "all"
	welcomeMessage = "Welcome to chat.\r\n" +
					 "Just enter text to send public message.\r\n" +
					 "Commands (without quotes):\r\n" +
					 "'\\to:<user> <message>' to send privately <message> to <user>.\r\n" +
					 "'\\quit' to leave chat and break connection.\r\n" +
					 "Currently there are %v user(s) is the chat%v.\r\n"
)

var (
	users = make(map[string]*user)
	n = 0
	mutex sync.Mutex
)

type user struct {
	username string
	conn     net.Conn
	input    chan message
}

type message struct {
	from string
	to   string
	string
}

func main() {
	l, err := net.Listen("tcp", ":2000")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf("%v connected", conn.RemoteAddr())
		
		fmt.Fprintf(conn, welcomeMessage, n, UserList())
		name, user, err := NewUser(conn)
		loop:
		for {
			switch {
			case err == nil:
				break loop
			case err.Error() == "User already exists":
				fmt.Fprintf(conn, "Error: this username is occuped\r\n")
			default:
				log.Println(err)
				fmt.Fprintf(conn, "Error reading username, try again.\r\n")
			} 
			name, user, err = NewUser(conn)
		}
		
		mutex.Lock()
		n++
		users[name] = user
		mutex.Unlock()
		go users[name].receive()
		go users[name].send()
		user.input <- message{system, name, fmt.Sprintf("Hello, %v. You can start chatting now.\r\n", name)}
	}
}

func NewUser(conn net.Conn) (string, *user, error) {
	fmt.Fprintf(conn, "Please, enter your username\r\n")
	username, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return username, nil, err
	}
	username = strings.TrimSpace(username)
	if _, present := users[username]; present {
		return username, nil, errors.New("User already exists")
	}
	log.Printf("%v joined the chat as %v\n", conn.RemoteAddr(), username)
	for _, user := range users {
		user.input <- message{system, all, fmt.Sprintf("%v entered chat\r\n", username)}
	}
	return username, &user{username, conn, make(chan message)}, nil
}

func (user *user) receive() {
	for {
		mes := <-user.input
		if mes.string == "closed" {
			return
		}
		fmt.Fprintf(user.conn, "from: %v, to: %v\r\n%v", mes.from, mes.to, mes.string)
	}
}

func (user *user) send() {
	for {
		if _, present := users[user.username]; !present {
			return
		}
		str, err := bufio.NewReader(user.conn).ReadString('\n')
		if err != nil {
			log.Println(err)
			user.input <- message{system, user.username, "Error occuped reading your message, try again."}
			continue
		}
		var mes message
		switch {
		case str[0] != '\\':
			mes = message{user.username, all, str}
		case strings.HasPrefix(str, "\\to:"):
			str := strings.SplitN(strings.TrimPrefix(str, "\\to:"), " ", 2)
			if _, present := users[str[0]]; !present {
				user.input <- message{system, user.username, fmt.Sprintf("Error: no such user as %v\r\n", strings.TrimSpace(str[0]))}
				continue
			}
			mes = message{user.username, str[0], str[1]}
		case strings.HasPrefix(str, "\\quit"):
			mes = message{system, all, fmt.Sprintf("User %v has left the chat\r\n", user.username)}
			user.input <- message{system, user.username, "closed"}
			log.Printf("%v (%v) has left the chat", user.username, user.conn.RemoteAddr())
			user.conn.Close()
			mutex.Lock()
			n--
			delete(users, user.username)
			mutex.Unlock()
		default:
			user.input <- message{system, user.username, fmt.Sprint("Unknown command\r\n")}
			continue
		}
		if mes.to == all {
			for _, u := range users {
				u.input <- mes
			}
		} else {
			users[mes.to].input <- mes
		}
	}
}

func UserList() string {
	str := ""
	for _, user := range users {
		str += ", " + user.username
	}
	str = strings.TrimPrefix(str, ", ")
	if n > 0 {
		str = ": " + str 
	}
	return str
}