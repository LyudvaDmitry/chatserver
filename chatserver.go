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

type Chatserver struct {
	userlist map[string]*user
	sync.Mutex
}

//Run initializes server and starts handling connections.
func (cs *Chatserver) Run() {
	cs.userlist = make(map[string]*user)
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
		name, err = cs.getStr(conn)
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
	cs.Send(message{system, all, fmt.Sprintf("%v entered chat\r\n", name)})
	u := &user{name, conn, make(chan message)}
	cs.Lock()
	cs.userlist[name] = u
	cs.Unlock()
	cs.handleUser(u)
	cs.Send(message{system, name, fmt.Sprintf("Hello, %v. You can start chatting now.\r\n", name)})
}

//handleUser runs goroutines implementing communication with user. They stop as soon
//as chatserver.Delete is called.
func (cs *Chatserver) handleUser(User *user) {
	//Messages for user
	go func() {
		for mes := range User.input {
			fmt.Fprintf(User.conn, "from: %v, to: %v\r\n%v", mes.from, mes.to, mes.string)
		}
	}()
	//Messages from user
	go func() {
		for {
			str, err := cs.getStr(User.conn)
			if err != nil {
				log.Println(err)
				cs.Delete(User.name)
				return
			}
			//Creating message and executing commands
			var mes message
			switch {
			case str[0] != '\\':
				mes = message{User.name, all, str + "\r\n"}
			case strings.HasPrefix(str, "\\to:"):
				comm := strings.SplitN(strings.TrimPrefix(str, "\\to:"), " ", 2)
				if cs.Get(comm[0]) == nil {
					cs.Send(message{system, User.name, fmt.Sprintf("Error: no such user as %v\r\n", comm[0])})
					continue
				}
				if len(comm) == 1 {
					comm = append(comm, "")
				}
				mes = message{User.name, comm[0], comm[1] + "\r\n"}
			case strings.HasPrefix(str, "\\quit"):
				cs.Delete(User.name)
				return
			default:
				cs.Send(message{system, User.name, fmt.Sprint("Unknown command\r\n")})
				continue
			}
			cs.Send(mes)
		}
	}()
}

//getStr reads string from given connection and parses it.
func (cs *Chatserver) getStr(conn net.Conn) (string, error) {
	str, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", err
	}
	str = strings.TrimSpace(str)
	runes := []rune(str)
	res := make([]rune, len(str))
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

//Delete deletes user from chatserver and closes its connection.
func (cs *Chatserver) Delete(name string) {
	cs.Lock()
	log.Printf("%v (%v) has left the chat", name, cs.userlist[name].conn.RemoteAddr())
	close(cs.userlist[name].input)
	cs.userlist[name].conn.Close()
	delete(cs.userlist, name)
	cs.Unlock()
	cs.Send(message{system, all, fmt.Sprintf("User %v has left the chat\r\n", name)})
}

//Get returns user with given username.
func (cs *Chatserver) Get(name string) *user {
	cs.Lock()
	defer cs.Unlock()
	return cs.userlist[name]
}

//List returns list of users currenty online as string.
func (cs *Chatserver) List() string {
	temp := make([]string, 0, cs.Len())
	cs.Lock()
	for _, user := range cs.userlist {
		temp = append(temp, user.name)
	}
	cs.Unlock()
	str := strings.Join(temp, ", ")
	if cap(temp) > 0 {
		str = ": " + str
	}
	return str
}

//Len returns current number of users.
func (cs *Chatserver) Len() int {
	cs.Lock()
	defer cs.Unlock()
	return len(cs.userlist)
}

//Send sends given message according to its 'from' and 'to' fields.
func (cs *Chatserver) Send(mes message) {
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
