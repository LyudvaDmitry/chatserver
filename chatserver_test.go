package chatserver

import (
//	"bufio"
	"testing"
	"fmt"
	"net"
	"log"
	"time"
	"sync"
//	"os"
)

func TestMain(t *testing.T) {
	var l *net.TCPListener
	go main(l)
	defer l.Close()
	var wg sync.WaitGroup
    wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			conn, err := net.Dial("tcp", "127.0.0.1:2000")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(conn, "bot%d\r\n", i)
			for j := 0; j < 10; j++ {
				time.Sleep(200 * time.Millisecond)
				//log.Printf("Sending message #%d from bot #%d", j, i)
				fmt.Fprintf(conn, "Message #%d from bot #%d\n", j, i)
			}
			time.Sleep(200 * time.Millisecond)
			fmt.Fprintf(conn, "\\quit\r\n")
			wg.Done()
		}(i)
	}
    wg.Wait()
	time.Sleep(2000 * time.Millisecond)
	//users.Lock()
	//log.Println(count)
	//users.Unlock()
}