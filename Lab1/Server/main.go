package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"

	"github.com/zh-five/golimit"
)

var current_conn int

func main() {
	// Get address and port number from command line
	addr := GetAddr()
	if addr == "-1" {
		fmt.Println("Address format error! Using default address: localhost:8080")
		addr = "localhost:8080"
	} else if addr == "-2" {
		fmt.Println("Using docker for server! Listening on all interfaces of port 8080")
		addr = "0.0.0.0:8080"
	}
	root := "./root"
	ListenAndServe(addr, root)
}

func ListenAndServe(address string, root string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Listen is err!: ", err)
	}
	defer listener.Close()
	fmt.Println("Listening on " + address)

	g := golimit.NewGoLimit(10) // 10 concurrent connections
	current_conn = 0
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept err!: ", err)
			continue
		}
		g.Add()
		current_conn++
		fmt.Println("Current connection number: ", current_conn)
		fmt.Println("Connection from ", conn.RemoteAddr().String())
		go HandleConnection(g, conn, root)
	}
}

func HandleConnection(g *golimit.GoLimit, conn net.Conn, root string) {
	// read from connection

	for {
		// read request
		br := bufio.NewReaderSize(conn, 50*1024*1024) // 50MB buffer
		request, err_readReq := http.ReadRequest(br)
		if err_readReq != nil {
			current_conn--
			g.Done()
			defer conn.Close()
			if err_readReq.Error() == "EOF" {
				fmt.Println("Connection closed by client")
				fmt.Println("Current connection number: ", current_conn)
				fmt.Println("--------------------------------------------------")
				return
			} else {
				fmt.Println("Current connection number: ", current_conn)
				fmt.Println("Request err:", err_readReq)
				// Return 400 Bad Request
				conn.Write([]byte("HTTP/1.1 400 Bad Request\r"))
				return
			}
		}

		// Set up response
		request.Response = new(http.Response)
		fmt.Println("Request Method:\n", request.Method) // "GET", "POST"
		fmt.Println("Request content:\n", request.URL)

		// Handle request with function handleRequest, only GET and POST. Other methods should return 405.
		var respCode int
		if request.Method == "GET" {
			respCode = GetHandler(request)
		} else if request.Method == "POST" {
			respCode = PostHandler(request)
		} else {
			respCode = UnsupportedMethodHandler(request)
		}
		request.Response.Write(conn)
		fmt.Println("Send response", respCode, "successfully!")
		defer request.Response.Body.Close()

	}
}
