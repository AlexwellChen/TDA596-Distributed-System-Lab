package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
)

func main() {
	fmt.Println("Starting the Proxy ...")

	proxy_addr := GetAddr()
	if proxy_addr == "-1" {
		fmt.Println("Address format error! Using default address: localhost:8081")
		proxy_addr = "localhost:8081"
	}

	listener, err := net.Listen("tcp", proxy_addr)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
	}
	defer listener.Close()
	fmt.Println("Start Listening on: " + proxy_addr + " ...")

	for {
		client, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accept client: ", err.Error())
		}
		// Read host addr from client
		var buffer [1024]byte
		host_addr, _ := client.Read(buffer[:])
		// Construct server connection
		host_addr_str := string(buffer[:host_addr])
		// Remove the last '\n'
		host_addr_str = host_addr_str[:len(host_addr_str)-1]
		server, _ := net.Dial("tcp", host_addr_str)
		go handleClientRequest(client, server)
	}
}

func handleClientRequest(client net.Conn, server net.Conn) {
	// server = FirstRequestHandler(client)
	fmt.Println("Client connected: ", client.RemoteAddr())
	for {
		if client == nil {
			return
		}
		var buffer [1024]byte
		client_request_msg, err := client.Read(buffer[:])
		if err != nil {
			if err == io.EOF {
				defer client.Close()
				defer server.Close()
				fmt.Println("*******************************************************")
				fmt.Println("-------------------- Proxy over here ------------------")
				fmt.Println("*******************************************************")
			} else {
				fmt.Println("Error reading client buffer:", err.Error())
			}
			return
		}

		//Construct client request
		client_request, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(buffer[:client_request_msg])))
		if err != nil {
			fmt.Println("Error reading client request:", err.Error())
			return
		}

		//Construct client_request's Response
		client_request.Response = new(http.Response)

		// If method is not GET, return 501
		if client_request.Method != "GET" {
			client_request.Response.StatusCode = http.StatusNotImplemented
			client_request.Response.Write(client)
			return
		}

		// Construct proxy to server request
		src_url := client_request.URL
		fmt.Println("src_url: ", src_url)
		server_request, err := http.NewRequest("GET", src_url.String(), nil)
		if err != nil {
			// Return internal server error
			client_request.Response.StatusCode = http.StatusInternalServerError
			client_request.Response.Write(client)
			fmt.Println("Error constructing server request:", err.Error())
			return
		}

		// Copy client request header to server request
		for k, v := range client_request.Header {
			server_request.Header[k] = v
		}

		// Send server request
		err = server_request.Write(server)
		if err != nil {
			// Return internal server error
			client_request.Response.StatusCode = http.StatusInternalServerError
			client_request.Response.Write(client)
			fmt.Println("Error writing server request:", err.Error())
			return
		}

		// Read server response
		server_response, err := http.ReadResponse(bufio.NewReader(server), server_request)
		if err != nil {
			// Return internal server error
			client_request.Response.StatusCode = http.StatusInternalServerError
			client_request.Response.Write(client)
			fmt.Println("Error reading server response:", err.Error())
			return
		}
		defer server_response.Body.Close()

		// Copy server response attributes to client response
		client_request.Response.StatusCode = server_response.StatusCode
		client_request.Response.ContentLength = server_response.ContentLength
		client_request.Response.Header = server_response.Header
		client_request.Response.Body = server_response.Body

		// Send client response
		err = client_request.Response.Write(client)
		if err != nil {
			fmt.Println("Error writing client response:", err.Error())
			return
		}

		defer client_request.Response.Body.Close()
	}

}
