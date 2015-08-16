//Package chatserver implements simple chatserver using TCP-socket.
//Telnet is expected to be used as client.
package chatserver

import (
	"bufio"
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

//Chatserver represents chatserver itself.
type Chatserver struct {
	userlist map[string]*user
	sync.RWMutex
}

//NewChatserver returns new Chatserver object.
func NewChatserver() *Chatserver {
	return &Chatserver{userlist: make(map[string]*user)}
}

//Run initializes server and starts handling connections.
func (cs *Chatserver) Run() {
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
		fmt.Fprintf(conn, welcomeMessage, cs.Len(), cs.List())
		cs.addUser(conn)
	}
}

//addUser requests new user to enter his username, creates user object,
//adds it to userlist and call handleUser().
func (cs *Chatserver) addUser(conn net.Conn) {
	//Trying to get right username till user give up.
	var name string
	var err error
	for {
		fmt.Fprintf(conn, "Please, enter your username\r\n")
		name, err = getStr(conn)
		if err != nil {
			fmt.Fprintf(conn, "Error reading username, try again.\r\n")
			continue
		}
		if cs.Get(name) != nil {
			fmt.Fprintf(conn, "Error: this username is occuped\r\n")
			continue
		}
		break
	}
	log.Printf("%v joined the chat as %v\n", conn.RemoteAddr(), name)
	cs.send(message{system, all, fmt.Sprintf("%v entered chat", name)})
	u := &user{name, conn, make(chan message)}
	cs.Lock()
	cs.userlist[name] = u
	cs.Unlock()
	cs.handleUser(u)
	cs.send(message{system, name, fmt.Sprintf("Hello, %v. You can start chatting now.", name)})
}

//handleUser runs goroutines implementing communication with user. They stop as soon
//as chatserver.Delete is called.
func (cs *Chatserver) handleUser(User *user) {
	//Messages for user
	go func() {
		for mes := range User.input {
			fmt.Fprintf(User.conn, "from: %v, to: %v\r\n%v\r\n", mes.from, mes.to, mes.string)
		}
	}()
	//Messages from user
	go func() {
		for {
			str, err := getStr(User.conn)
			if err != nil {
				log.Println(err)
				cs.Delete(User.name)
				return
			}
			//Creating message and executing commands
			var mes message
			switch {
			case str[0] != '\\':
				mes = message{User.name, all, str}
			case strings.HasPrefix(str, "\\to:"):
				comm := strings.SplitN(strings.TrimPrefix(str, "\\to:"), " ", 2)
				if len(comm) == 1 {
					comm = append(comm, "")
				}
				mes = message{User.name, comm[0], comm[1]}
			case strings.HasPrefix(str, "\\quit"):
				cs.Delete(User.name)
				return
			default:
				cs.send(message{system, User.name, fmt.Sprint("Unknown command")})
				continue
			}
			cs.send(mes)
		}
	}()
}

//Delete deletes user from chatserver and closes its connection.
func (cs *Chatserver) Delete(name string) {
	cs.Lock()
	log.Printf("%v (%v) has left the chat", name, cs.userlist[name].conn.RemoteAddr())
	close(cs.userlist[name].input)
	cs.userlist[name].conn.Close()
	delete(cs.userlist, name)
	cs.Unlock()
	cs.send(message{system, all, fmt.Sprintf("User %v has left the chat", name)})
}

//Get returns user with given username.
func (cs *Chatserver) Get(name string) *user {
	cs.RLock()
	defer cs.RUnlock()
	return cs.userlist[name]
}

//List returns list of users currenty online as string.
func (cs *Chatserver) List() string {
	temp := make([]string, 0, cs.Len())
	cs.RLock()
	for _, user := range cs.userlist {
		temp = append(temp, user.name)
	}
	cs.RUnlock()
	str := strings.Join(temp, ", ")
	if cap(temp) > 0 {
		str = ": " + str
	}
	return str
}

//Len returns current number of users.
func (cs *Chatserver) Len() int {
	cs.RLock()
	defer cs.RUnlock()
	return len(cs.userlist)
}

//send sends given message according to its 'from' and 'to' fields.
func (cs *Chatserver) send(mes message) {
	if cs.Get(mes.to) == nil {
		cs.send(message{system, mes.from, fmt.Sprintf("Error: no such user as %v", mes.to)})
		return
	}
	cs.Lock()
	defer cs.Unlock()
	log.Printf("Mes(%v->%v): %v", mes.from, mes.to, mes.string)
	if mes.to == all {
		for _, u := range cs.userlist {
			u.input <- mes
		}
	} else {
		cs.userlist[mes.to].input <- mes
	}
}

type user struct {
	name  string
	conn  net.Conn
	input chan message
}

type message struct {
	from string
	to   string
	string
}

//getStr reads string from given connection and parses it.
func getStr(conn net.Conn) (string, error) {
	str, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", err
	}
	str = strings.TrimSpace(str)
	runes := []rune(str)
	res := make([]rune, len(runes))
	i := 0
	for _, char := range runes {
		switch char {
		case '\b':
			i--
		default:
			res[i] = char
			i++
		}
	}
	res = res[:i]
	return string(res), nil
}