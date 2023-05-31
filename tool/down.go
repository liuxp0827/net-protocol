package main

import (
	"fmt"
	"github.com/liuxp0827/net-protocol/config"

	"github.com/liuxp0827/net-protocol/protocol/link/tuntap"
)

func main() {
	//关闭网卡
	if err := tuntap.DelTap(config.NicName); err != nil {
		fmt.Println(err)
		return
	}
}
