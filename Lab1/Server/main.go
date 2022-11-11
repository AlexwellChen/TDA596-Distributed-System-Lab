package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
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

func getHandler(r *http.Request) {
	fmt.Println("Invoke GET Handler")
	response := r.Response

	url := r.URL.Path
	pwd, _ := os.Getwd()
	url = pwd + url

	// Check if file exists
	s, err := os.Stat(url)
	if err != nil {
		fmt.Println("Status error: ", err)
		response.StatusCode = http.StatusNotFound
		response.Body = ioutil.NopCloser(strings.NewReader("Resource not found"))
		return
	}

	// Check if file or directory could be read
	file, err := os.Open(url)
	if err != nil {
		fmt.Println("Open file or directory error: ", err)
		// Return resource not found
		response.StatusCode = http.StatusNotFound
		response.Body = ioutil.NopCloser(strings.NewReader("Resource not found"))
		return
	}
	defer file.Close()

	// Check if it is a directory
	if s.IsDir() {
		// a directory
		file_info, err := file.Readdir(-1)
		if err != nil {
			fmt.Println("Read file error!")
			// Return internal server error
			response.StatusCode = http.StatusInternalServerError
			response.Body = ioutil.NopCloser(strings.NewReader("Internal server error"))
			return
		}

		// file_info to string
		var file_info_str string
		for _, file := range file_info {
			file_info_str += file.Name() + " "
		}
		response.StatusCode = http.StatusOK
		response.Body = ioutil.NopCloser(strings.NewReader(file_info_str))
	} else {
		// a file
		bytes, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Println("Read file error!")
			// Return internal server error
			response.StatusCode = http.StatusInternalServerError
			response.Body = ioutil.NopCloser(strings.NewReader("Internal server error"))
			return
		}
		response.StatusCode = http.StatusOK
		response.Body = ioutil.NopCloser(strings.NewReader(string(bytes)))
	}

}

func postHandler(r *http.Request) {
	fmt.Println("Invoke POST Handler")
	url := r.URL.Path
	fmt.Println("URL: ", url)
}

func unsupportedMethodHandler(r *http.Request) {
	response := r.Response
	response.StatusCode = http.StatusMethodNotAllowed
	response.Body = ioutil.NopCloser(strings.NewReader("Method not allowed"))
	fmt.Println("Unsupported method!")
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
	// read from connection
	msg, err := conn.Read(buffer)
	if err != nil {
		// handle error
		fmt.Println("connection err!:", err)
		return
	}
	// print message
	fmt.Println("Connection from ", conn.RemoteAddr().String())

	// msg to request
	request_str := string(buffer[:msg])
	br := bufio.NewReader(strings.NewReader(request_str))
	request, err_cnn := http.ReadRequest(br)

	if err_cnn != nil {
		fmt.Println("Request err:", err)
		return
	}

	request.Response = new(http.Response)

	fmt.Println("Request Method:\n", request.Method) // "GET", "POST"
	fmt.Println("Request content:\n", request.URL)

	// Handle request with function handleRequest, only GET and POST. Other methods should return 405.

	if request.Method == "GET" {
		getHandler(request)
	} else if request.Method == "POST" {
		postHandler(request)
	} else {
		unsupportedMethodHandler(request)
	}
	request.Response.Write(conn)
	fmt.Println("Send response successfully!")
	defer request.Response.Body.Close()
	fmt.Println("--------------------------------------------------")
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
