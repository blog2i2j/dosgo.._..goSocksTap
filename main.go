package main

import (
	"flag"
	"fmt"

	"github.com/dosgo/goSocksTap/socksTap"
)

func main() {

	var sock5Addr = ""
	flag.StringVar(&sock5Addr, "sock5Addr", "127.0.0.1:10808", " socks5 addr ")
	var udpProxy = false
	flag.BoolVar(&udpProxy, "udpProxy", false, "use udpProxy ")
	flag.Parse()
	var _socksTap = socksTap.SocksTap{}
	fmt.Printf("sock5Addr:%s\r\n", sock5Addr)
	fmt.Printf("udpProxy:%v\r\n", udpProxy)
	_socksTap.Start(sock5Addr, "xxx.com", udpProxy)
	select {}
}
