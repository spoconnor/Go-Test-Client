package main

import (
	"bufio"
	"flag"
	"log"
	"os"

	logging "github.com/spoconnor/Go-Common-Code/logging"
	client "github.com/spoconnor/Go-Test-Client/Client"
)

var addr = flag.String("ws", "127.0.0.1:8080", "websocket address eg '192.168.158.129:8080'")
var proxy = flag.String("proxy", "", "proxy address eg 'http://localhost:8888'")

// main entry to app
func main() {
	logging.SetupLog("test-client", "info", "output.txt", true)
	log.Println("Starting test-client")

	flag.Parse()
	log.SetFlags(0)

	c := client.NewClient(*addr, *proxy)
	c.RegisterService(new(client.Service1), "Test Service")
	go c.Start()

	log.Println("Press any key to exit")
	reader := bufio.NewReader(os.Stdin)
	_, _, _ = reader.ReadRune()
	log.Println("Done")
}
