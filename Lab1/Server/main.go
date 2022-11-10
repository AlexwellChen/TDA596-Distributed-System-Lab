package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
)

const (
	Limit  = 10 // Upper limit of concurrent connections
	Weight = 1  // Weight of each connection
)

// Get port number from command line
func getPort() int {
	args := os.Args
	if len(args) != 2 {
		fmt.Println("Arguments length error!")
		return -1
	}
	port, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Println("Port number error!")
		return -1
	}
	return port
}

func ListenAndServe(address string, root string) error {
	// max_delay := 2 // seconds
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Listen is err!: ", err)
	}
	defer listener.Close()
	fmt.Println("Listening on " + address)
	//Todo: Add concurrency control here, maxmum 10 connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept is err!: ", err)
		}
		go handleConnection(conn, root)
	}
}

func handleConnection(conn net.Conn, root string) {
	//Create an empty buffer
	buffer := make([]byte, 1024)
	defer conn.Close()
	for {
		// read from connection
		msg, err := conn.Read(buffer)
		fmt.Println("Read from connection: ", string(buffer[:msg]))
		if err != nil {
			// handle error
			if err.Error() == "EOF" {
				fmt.Println("Connection closed by client")
			} else {
				fmt.Println("connection err!:", err)
			}
			return
		}
		// print message
		fmt.Println(conn.RemoteAddr().String())

		// msg to request
		request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buffer[:msg])))
		fmt.Println("Request content:\n", request)

		// TODO: handle request with function handleRequest, only GET and POST. Other methods should return 405.
		// TODO: how to create a response writer?
		w := http.ResponseWriter{}
		response := handleRequest(w, request, root)
		fmt.Println("Response content:\n", response)

		/*
			Todo:
				send response to client
				if content is a directory, send a string of files in it
				if content is a file, use POST method to send it
		*/
		// response to connection
		conn.Write([]byte("Test response ack"))
		fmt.Println("--------------------")
	}
}

func handleRequest(w http.ResponseWriter, request *http.Request, root string) {
	w.Header().Set("Content-Type", "text/plain")
	if request.Method == "GET" {
		//Return the content of the file
		path := request.URL.Path
		// open file and read content to buffer
		_, err := os.Open(root + path)
		if err != nil {
			fmt.Println("Open file error: ", err)
			w.WriteHeader(http.StatusBadRequest)
		} else {
			fmt.Println("File exists")
			w.WriteHeader(http.StatusOK)
			// w.Write([]byte(file))
		}
	} else if request.Method == "POST" {

	} else {
		// Response "Not Implemented" (501)
		fmt.Println("invalid method")
	}
}
func main() {
	port := getPort()
	if port == -1 {
		fmt.Println("Please state port number!")
		return
	}
	addr := "127.0.0.1:" + strconv.Itoa(port)
	root := "./root"
	ListenAndServe(addr, root)
}
