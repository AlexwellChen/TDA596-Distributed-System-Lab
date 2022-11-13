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
	var buffer [1024]byte
	client_request_msg, err := client.Read(buffer[:])
	if err != nil {
		fmt.Println("Error reading client buffer:", err.Error())
		return
	}

	var method, host, address string
	fmt.Sscanf(string(buffer[:bytes.IndexByte(buffer[:], '\n')]), "%s%s", &method, &host)
	hostPortURL, err := url.Parse(host)
	if err != nil {
		fmt.Println("Error parsing host:", err.Error())
		return
	}

	//http访问
	if !strings.Contains(hostPortURL.Host, ":") {
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
	defer server.Close()

	// Forwarding client request to server
	server.Write(buffer[:client_request_msg])

	// Data transfer between client and server
	go io.Copy(server, client)
	io.Copy(client, server)
	// Todo: Close connection when client is closed

	// Unable to reach here
	client.Close()
	fmt.Println("Connection closed")
	fmt.Println("--------------------------------------------------")
}
