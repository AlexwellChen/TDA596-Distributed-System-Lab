package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func main() {
	fmt.Println("Starting Proxy ...")
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
	}
	defer listener.Close()
	fmt.Println("************************************")
	fmt.Println("Proxy running on: ", listener.Addr())
	fmt.Println("************************************")
	for {
		fmt.Println("Waiting for a connection ...")
		client, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accept client: ", err.Error())
		}

		fmt.Println("Client address: ", client.RemoteAddr())
		go handleClientRequest(client)
	}
}

func handleClientRequest(client net.Conn) {
	if client == nil {
		return
	}
	var buffer [1024]byte
	for {
		client_request, client_err := client.Read(buffer[:])
		if client_err != nil {
			if client_err.Error() == "EOF" {
				fmt.Println("Client closed connection")
			} else {
				fmt.Println("Error reading client buffer:", client_err.Error())
			}
			client.Close()
			return
		}
		var method, host, address string
		// fmt.Sscanf(string(buffer[:bytes.IndexByte(buffer[:], '\n')]), "%s%s", &method, &host)

		// Client request to proxy

		client_request_proxy, err_cnn := http.ReadRequest(bufio.NewReader(strings.NewReader(string(buffer[:client_request]))))
		if err_cnn != nil {
			fmt.Println("Request err:", err_cnn)
			return
		}

		// Proxy request to Server
		var proxy_request_server *http.Request
		proxy_request_server, err := http.NewRequest(method, host, client_request_proxy.Body)

		hostPortURL, err := url.Parse(host)
		if err != nil {
			fmt.Println("Error parsing host:", err.Error())
			return
		}

		// Get remote address
		if strings.Index(hostPortURL.Host, ":") == -1 {
			//host不带端口， 默认8080
			address = hostPortURL.Host + ":8080"
		} else {
			address = hostPortURL.Host
		}
		fmt.Println("Host address:", address)

		//Connect to the remote server
		server, err := net.Dial("tcp", address)
		if err != nil {
			fmt.Println("Error dialing server:", err.Error())
			return
		}
		fmt.Println("Proxying request to server on: ", server.LocalAddr())

		// Send proxy request to server
		proxy_request_server.Write(server)

		// Send response to client
		var response_buffer [4096]byte
		resp_msg, err := server.Read(response_buffer[:])
		if err != nil {
			fmt.Println("Error reading server buffer:", err.Error())
			return
		}
		client_request_proxy.Response.StatusCode = proxy_request_server.Response.StatusCode
		client_request_proxy.Response.ContentLength = int64(resp_msg)
		client_request_proxy.Response.Body = ioutil.NopCloser(bytes.NewBuffer(response_buffer[:resp_msg]))

		defer server.Close()
		fmt.Println("--------------------------------------------------")
	}

}
