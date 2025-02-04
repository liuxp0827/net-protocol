package main

import (
	"fmt"
	"github.com/liuxp0827/net-protocol/config"

	"github.com/liuxp0827/net-protocol/pkg/logging"
	"github.com/liuxp0827/net-protocol/protocol/application/http"
)

func init() {
	logging.Setup()
}
func main() {
	serv := http.NewHTTP(config.NicName, "192.168.1.0/24", "192.168.1.1", "9502")
	serv.HandleFunc("/", func(request *http.Request, response *http.Response) {
		fmt.Println("hell0 ----------------------")
		response.End("hello")
	})
	serv.ListenAndServ()
}
