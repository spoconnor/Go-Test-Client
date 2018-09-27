package client

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spoconnor/Go-Test-Client/JsonRpc"
)

type Client struct {
	addr      string
	proxy     string
	server    JsonRpc.Server
	Listening bool
	ClientKey string
}

// MyProxy sets up the proxy address and port
func (c *Client) myProxy(req *http.Request) (*url.URL, error) {
	if c.proxy == "" {
		return nil, nil
	}
	url, err := url.Parse(c.proxy)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return url, nil
}

func NewClient(addr string, proxy string) *Client {
	c := &Client{
		addr:      addr,
		proxy:     proxy,
		Listening: false,
	}
	c.server = *JsonRpc.NewServer()
	return c
}

func (c *Client) RegisterService(receiver interface{}, namespace string) {
	c.server.RegisterService(receiver, namespace)
}

// main entry to app
func (c *Client) Start() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: c.addr, Path: "/"}
	log.Printf("connecting to %s", u.String())

	var dialer = &websocket.Dialer{
		Proxy:            c.myProxy,
		HandshakeTimeout: 45 * time.Second,
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	go c.RunLoop(conn)
	c.Listening = true

	// TODO - tidy up
	for true {
		time.Sleep(1000 * time.Millisecond)
	}
}

func (c *Client) RunLoop(conn *websocket.Conn) {
	done := make(chan struct{})
	defer close(done)
	for {
		_, msgBody, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		log.Printf("recv: %s", msgBody)
		jsonBody := string(msgBody)
		if jsonBody == "ClientKeyPlease" {
			c.ClientKey = fmt.Sprintf("Client%d", os.Getpid())
			response := fmt.Sprintf("ClientKey:%s\nChallenge:Wibble", c.ClientKey)
			log.Printf("Sending client key: %s", response)
			err = conn.WriteMessage(websocket.BinaryMessage, []byte(response))
		} else {
			log.Printf("Receiving message...")
			request := &JsonRpc.Request{Body: jsonBody}
			response := &JsonRpc.Response{}
			c.server.ServeRequest(request, response)
			err = conn.WriteMessage(websocket.TextMessage, []byte(response.Body))
		}
		if err != nil {
			log.Println("write error:", err)
			return
		}
	}
}
