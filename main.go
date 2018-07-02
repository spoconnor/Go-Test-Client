package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"pharos/test-client/JsonRpc"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "192.168.158.129:8080", "http service address")
var key = flag.String("key", "Alice123", "Connection key")

func MyProxy(req *http.Request) (*url.URL, error) {
	url, err := url.Parse("http://localhost:8888")
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return url, nil
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	server := JsonRpc.NewServer()
	server.RegisterService(new(Service1), "")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/"}
	log.Printf("connecting to %s", u.String())

	var dialer = &websocket.Dialer{
		Proxy:            MyProxy,
		HandshakeTimeout: 45 * time.Second,
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, msgBody, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", msgBody)
			jsonBody := string(msgBody)
			if jsonBody == "RoutingKeyPlease" {
				//response := fmt.Sprintf("RoutingKey:Alice%d\nChallenge:Wibble", os.Getpid())
				response := fmt.Sprintf("RoutingKey:%s\nChallenge:Wibble", *key)
				log.Printf("Sending routing key: %s", response)
				err = conn.WriteMessage(websocket.BinaryMessage, []byte(response))
			} else {
				log.Printf("Receiving message...")
				request := &JsonRpc.Request{Body: jsonBody}
				response := &JsonRpc.Response{}
				server.ServeRequest(request, response)
				err = conn.WriteMessage(websocket.TextMessage, []byte(response.Body))
			}
			if err != nil {
				log.Println("write error:", err)
				return
			}
		}
	}()

	log.Println("Press any key to exit")
	reader := bufio.NewReader(os.Stdin)
	_, _, err = reader.ReadRune()
	log.Println("Done")
}
