package chatserver

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

const (
	system         = "system"
	all            = "all"
	welcomeMessage = "Welcome to chat.\r\n" +
		"Just enter text to send public message.\r\n" +
		"Commands (without quotes):\r\n" +
		"'\\to:<user> <message>' to send privately <message> to <user>.\r\n" +
		"'\\quit' to leave chat and break connection.\r\n" +
		"Currently there are %v user(s) is the chat%v.\r\n"
)

var (
	users Users
	//k = 0
	//count = make(map[string]int)
)

type Users struct {
	list map[string]*user
	sync.Mutex
}

type user struct {
	name string
	conn     net.Conn
	input    chan message
}

//Quit closes connection, removes user from list and informs users that
//given user has left the chat.
func (User user) quit() {
	log.Printf("%v (%v) has left the chat", User.name, User.conn.RemoteAddr())
	User.conn.Close()
	users.Lock()
	delete(users.list, User.name)
	users.Unlock()
	message{system, all, fmt.Sprintf("User %v has left the chat\r\n", User.name)}.send()
}

type message struct {
	from string
	to   string
	string
}

//message.send sends message to the according channels.
func (mes message) send() {
	users.Lock()
	if mes.to == all {
		for _, u := range users.list {
			u.input <- mes
		}
	} else {
		users.list[mes.to].input <- mes
	}
	//k++
	//count[mes.from]++
	users.Unlock()
	//log.Printf("(%v) %v to %v: %v", k, mes.from, mes.to, strings.TrimSpace(mes.string))
	log.Printf("%v to %v: %v", mes.from, mes.to, strings.TrimSpace(mes.string))
}

func main(l net.Listener) {
	users.list = make(map[string]*user)
	l, err := net.Listen("tcp", ":2000")
	if err != nil {
		log.Fatal(err)
	}
	//Close l
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err, "1")
			continue
		}
		log.Printf("%v connected", conn.RemoteAddr())

		users.Lock()
		fmt.Fprintf(conn, welcomeMessage, len(users.list), users.UserList())
		users.Unlock()
		Name, User, err := NewUser(conn)
		//Trying to get right username till user give up.
		for err != nil {
			switch {
			case err.Error() == "User already exists":
				fmt.Fprintf(conn, "Error: this username is occuped\r\n")
			default:
				log.Println(err, "2")
				fmt.Fprintf(conn, "Error reading username, try again.\r\n")
			}
			Name, User, err = NewUser(conn)
		}

		users.Lock()
		users.list[Name] = User
		go users.list[Name].receive()
		go users.list[Name].send()
		users.Unlock()
		message{system, Name, fmt.Sprintf("Hello, %v. You can start chatting now.\r\n", Name)}.send()
	}
}

//NewUser returns user, its name and error, if any. 
func NewUser(conn net.Conn) (string, *user, error) {
	//Can't use send() or receive() here as there is no user for now.
	fmt.Fprintf(conn, "Please, enter your username\r\n")
	name, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return name, nil, err
	}
	name = backspaceFighter(strings.TrimSpace(name))
	users.Lock()
	_, present := users.list[name];
	users.Unlock()
	if present {
		return name, nil, errors.New("User already exists")
	}
	log.Printf("%v joined the chat as %v\n", conn.RemoteAddr(), name)
	message{system, all, fmt.Sprintf("%v entered chat\r\n", name)}.send()
	return name, &user{name, conn, make(chan message)}, nil
}

//Send waits for messages to be sent to the user.input channel
//and send it to user using user.conn.
func (User *user) send() {
	for mes := range User.input {
		log.Printf("from: %v, to: %v\r\n%v", mes.from, mes.to, mes.string)
		fmt.Fprintf(User.conn, "from: %v, to: %v\r\n%v", mes.from, mes.to, mes.string)
	}
}

//Receive receives messages from user and parses it. Then it send message or
//execute command depends on the message content.
func (User *user) receive() {
	for {
		//Reading and parsing
		str, err := bufio.NewReader(User.conn).ReadString('\n')
		if err != nil {
			log.Println(err)
			User.quit()
			return
		}
		str = backspaceFighter(str)
		log.Print(User.name + ": " + str)
		//users.Lock()
		//k--
		//users.Unlock()
		//Creating message and executing commands
		var mes message
		switch {
		case str[0] != '\\':
			mes = message{User.name, all, str}
		case strings.HasPrefix(str, "\\to:"):
			str := strings.SplitN(strings.TrimPrefix(str, "\\to:"), " ", 2)
			users.Lock()
			_, present := users.list[str[0]]
			users.Unlock()
			if !present {
				message{system, User.name, fmt.Sprintf("Error: no such user as %v\r\n", strings.TrimSpace(str[0]))}.send()
				continue
			}
			mes = message{User.name, str[0], str[1]}
		case strings.HasPrefix(str, "\\quit"):
			User.quit()
			return
		default:
			message{system, User.name, fmt.Sprint("Unknown command\r\n")}.send()
			continue
		}
		log.Print(User.name + ":(almost there) " + str)
		mes.send()
	}
}

//Userlist returns list of users currently online.
func (users *Users) UserList() string {
	temp := make([]string, 0, len(users.list))
	for _, user := range users.list {
		temp = append(temp, user.name)
	}
	str := strings.Join(temp, ", ")
	if cap(temp) > 0 {
		str = ": " + str
	}
	return str
}

//backspaceFighter "executes" backspaces.
func backspaceFighter (input string) string {
	in := []byte(input)
	res := make([]byte, len(input))
	i := 0
	for _, char := range in {
		switch char {
		case '\b':
			i--
		default:
			res[i] = char
			i++
		}
	}
	res = res[:i]
	return string(res)
}