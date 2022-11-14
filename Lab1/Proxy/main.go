package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
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
	fmt.Println("Start Listening ...")
	for {

		client, err := listener.Accept()

		if err != nil {
			fmt.Println("Error accept client: ", err.Error())
		}
		go handleClientRequest(client)
	}
}

func handleClientRequest(client net.Conn) {
	for {
		if client == nil {
			return
		}
		var buffer [1024]byte
		client_request_msg, err := client.Read(buffer[:])
		if err != nil {
			if err == io.EOF {
				fmt.Println("Client closed connection")
				defer client.Close()
				fmt.Println("*******************************************************")
				fmt.Println("--------------------Proxy over here--------------------")
				fmt.Println("*******************************************************")
			} else {
				fmt.Println("Error reading client buffer:", err.Error())
			}
			return
		}

		var method, host, address string
		fmt.Sscanf(string(buffer[:bytes.IndexByte(buffer[:], '\n')]), "%s%s", &method, &host)

		//Construct client request
		client_request, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(buffer[:client_request_msg])))
		if err != nil {
			fmt.Println("Error reading client request:", err.Error())
			return
		}

		//Construct client_request's Response
		client_request.Response = new(http.Response)

		// If method is not GET, return 501
		if method != "GET" {
			client_request.Response.StatusCode = http.StatusNotImplemented
			client_request.Response.Write(client)
			return
		}

		hostPortURL, err := url.Parse(host)
		if err != nil {
			// Return internal server error
			client_request.Response.StatusCode = http.StatusInternalServerError
			client_request.Response.Write(client)
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
			// Return internal server error
			client_request.Response.StatusCode = http.StatusInternalServerError
			client_request.Response.Write(client)
			fmt.Println("Error dialing server:", err.Error())
			return
		}
		defer server.Close()

		// Construct proxy to server request
		server_request, err := http.NewRequest("GET", host, nil)
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

		// Copy server response body to client response
		client_request.Response.StatusCode = server_response.StatusCode
		content, err := ioutil.ReadAll(server_response.Body)

		// fmt.Println("------------------Content test------------------")
		// fmt.Println("content:", string(content))

		if err != nil {
			// Return internal server error
			client_request.Response.StatusCode = http.StatusInternalServerError
			client_request.Response.Write(client)
			fmt.Println("Error reading server response body:", err.Error())
			return
		}
		client_request.Response.ContentLength = int64(len(content))
		client_request.Response.Body = ioutil.NopCloser(strings.NewReader(string(content)))
		// Send client response
		err = client_request.Response.Write(client)
		if err != nil {
			fmt.Println("Error writing client response:", err.Error())
			return
		}

		defer client_request.Response.Body.Close()
	}

}
