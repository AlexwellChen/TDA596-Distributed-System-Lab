package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"

	"golang.org/x/sync/semaphore"
)

const (
	Limit  = 1 // Upper limit of concurrent connections
	Weight = 1 // Weight of each connection
)

// global variable semaphore
var sem = semaphore.NewWeighted(Limit)

func main() {
	// Get address and port number from command line
	addr := GetAddr()
	if addr == "-1" {
		fmt.Println("Address format error! Using default address: localhost:8080")
		addr = "localhost:8080"
	}
	root := "./root"
	ListenAndServe(addr, root)
}

func ListenAndServe(address string, root string) error {
	// max_delay := 2 // seconds
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Listen is err!: ", err)
	}
	// defer listener.Close()
	fmt.Println("Listening on " + address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept err!: ", err)
			continue
		}
		valid := sem.TryAcquire(Weight)
		if !valid {
			fmt.Println("Semaphore full! Connection is rejected!")
			conn.Close()
			continue
		} else {
			fmt.Println("Semaphore acquired!")
			fmt.Println("Connection from ", conn.RemoteAddr().String())
			go HandleConnection(conn, root)
		}
	}
}

func HandleConnection(conn net.Conn, root string) {
	// read from connection
	defer sem.Release(Weight)
	for {
		// read request
		br := bufio.NewReaderSize(conn, 50*1024*1024) // 50MB buffer
		request, err_readReq := http.ReadRequest(br)
		defer conn.Close()

		if err_readReq != nil {
			if err_readReq.Error() == "EOF" {
				fmt.Println("Connection closed by client")
				fmt.Println("--------------------------------------------------")
				return
			} else {
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
