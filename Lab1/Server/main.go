package main

import (
	"fmt"
	"net"
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
	buffer := make([]byte, 1024)
	defer conn.Close()
	for {
		// read from connection
		msg, err := conn.Read(buffer)

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
		request := string(buffer[:msg])
		fmt.Println("Request content:\n", request)

		// Todo: handle request with function handleRequest, only GET and POST. Other methods should return 405.
		// response := handleRequest(request, root)

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

func main() {
	port := getPort()
	if port == -1 {
		return
	}
	addr := "127.0.0.1:" + strconv.Itoa(port)
	root := "./root"
	ListenAndServe(addr, root)
}
