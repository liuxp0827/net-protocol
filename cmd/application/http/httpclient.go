package main

import (
	"fmt"
	"github.com/liuxp0827/net-protocol/pkg/logging"
	"github.com/liuxp0827/net-protocol/protocol/application/http"
)

func init() {
	logging.Setup()

}
func main() {
	cli, err := http.NewClient("http://10.0.2.15:8080/test")
	if err != nil {
		panic(err)
		return
	}
	cli.SetMethod("GET")
	cli.SetData("test")
	res, err := cli.GetResult()
	fmt.Println(res)

}
