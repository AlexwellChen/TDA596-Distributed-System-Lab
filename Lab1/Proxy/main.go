package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
)

func main() {
	fmt.Println("Starting the Proxy ...")
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
	}
	defer listener.Close()
	for {
		fmt.Println("Waiting for a connection ...")
		client, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accept client: ", err.Error())
		}
		go handleClientRequest(client)
	}
}

func handleClientRequest(client net.Conn) {
	if client == nil {
		return
	}
	defer client.Close()
	var buffer [1024]byte
	msg, err := client.Read(buffer[:])
	if err != nil {
		fmt.Println("Error reading client buffer:", err.Error())
		return
	}
	defer client.Close()
	var method, host, address string
	fmt.Sscanf(string(buffer[:bytes.IndexByte(buffer[:], '\n')]), "%s%s", &method, &host)
	hostPortURL, err := url.Parse(host)
	if err != nil {
		fmt.Println("Error parsing host:", err.Error())
		return
	}

	//http访问
	if strings.Index(hostPortURL.Host, ":") == -1 {
		//host不带端口， 默认8080
		address = hostPortURL.Host + ":8080"
	} else {
		address = hostPortURL.Host
	}

	//获得了请求的host和port，就开始拨号吧
	server, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Error dialing server:", err.Error())
		return
	}
	if method == "CONNECT" {
		fmt.Fprint(client, "HTTP/1.1 200 Connection established\r\n")
	} else {
		server.Write(buffer[:msg])
	}
	//进行转发
	go io.Copy(server, client)
	io.Copy(client, server)
	fmt.Println("Connection closed")
	fmt.Println("--------------------------------------------------")
}
