package chatserver

import (
	"testing"
	"fmt"
	"net"
	"log"
	"time"
	"sync"
	"math/rand"
)

const (
	botnum = 10
	mesnum = 10
)

func TestRun(t *testing.T) {
	var cs Chatserver
	go cs.Run()
	var wg sync.WaitGroup
    wg.Add(botnum)
	for i := 0; i < botnum; i++ {
		go func(i int) {
			conn, err := net.Dial("tcp", "127.0.0.1:2000")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(conn, "bot%d\r\n", i)
			for j := 0; j < mesnum; j++ {
				time.Sleep(100 * time.Millisecond)
				fmt.Fprintf(conn, "Message #%d from bot #%d\n", j, i)
			}
    		r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for j := 0; j < mesnum; j++ {
				k := r.Intn(botnum)
				time.Sleep(100 * time.Millisecond)
				fmt.Fprintf(conn, "\\to:bot%d Message #%d from bot #%d to bot #%d\n", k, j, i, k)
			}
			time.Sleep(100 * time.Millisecond)
			fmt.Fprintf(conn, "\\quit\r\n")
			wg.Done()
		}(i)
	}
    wg.Wait()
	time.Sleep(200 * time.Millisecond)
}