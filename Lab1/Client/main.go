package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

func main() {
	// user input server address
	fmt.Println("Please enter server address:")
	reader := bufio.NewReader(os.Stdin)
	// server, _ := reader.ReadString('\n')
	server := "localhost:8080"
	//fmt.Println(strings.TrimSpace(server))
	server = strings.TrimSpace(server)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", server)

	if err != nil {
		fmt.Println("Fatal error: ", err)
		os.Exit(1)
	}

	// Connect to server
	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	if err != nil {
		// if connection was refused, conn does not exist and gives a nil pointer error
		// fmt.Println(conn.RemoteAddr().String(), os.Stderr, "Fatal error:", err)
		fmt.Println("Fatal error:", err)
		os.Exit(1)
	}
	fmt.Println("connection success")
	for {
		//repeat send request until user input "exit"
		//Ask user for input request resource and method?
		fmt.Println("Please enter request method:") //GET POST
		method, _ := reader.ReadString('\n')
		method = strings.TrimSpace(method)
		fmt.Println("Please enter request resource root:") //root
		resource, _ := reader.ReadString('\n')
		//resource = strings.TrimSpace(resource)
		resource = "/" + strings.TrimSpace(resource)
		fmt.Println("Please enter request file:") // file name
		fileName, _ := reader.ReadString('\n')
		fileName = strings.TrimSpace(fileName)
		//if fileName == "" {
		//	resourcePath := resource
		//	sender(conn, method, resourcePath, fileName)
		//} else {
		//	resourcePath := resource + "/" + fileName
		//	sender(conn, method, resourcePath, fileName)
		//}
		resourcePath := resource + "/" + fileName

		sender(conn, method, resourcePath, fileName)
		fmt.Println("send over")
	}
	/*	method := "GET"
		file := "2.jpg"
		resource := "/root/" + file
		sender(conn, method, resource, file)
		//sender(conn, method, resource)
		fmt.Println("send over")*/

}

func sender(conn *net.TCPConn, method string, resource string, fileName string) {
	host_addr := conn.RemoteAddr().String()

	url := "http://" + host_addr + resource
	fmt.Println("url:", url)

	// Create a new request
	request, _ := http.NewRequest(method, url, nil)
	err := request.Write(conn)
	if err != nil {
		fmt.Println(conn.RemoteAddr().String(), " Error: ", err)
		os.Exit(1)
	}
	// fmt.Println("current request:", request)
	// Read response from connection
	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, request)
	if err != nil {
		fmt.Println("Error reading response:", err)
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusInternalServerError:
		fmt.Println("500 Internal Server Error")
		break
	case http.StatusNotImplemented:
		fmt.Println("501 Not Implemented")
		break
	case http.StatusBadGateway:
		fmt.Println("502 Bad Gateway")
		break
	case http.StatusBadRequest:
		fmt.Println("400 Bad Request")
		break
	case http.StatusOK:
		fmt.Println("200 OK")
		if method == "GET" {
			if fileName == "" {
				fmt.Println("The files in the directory have been listed below:")
				_, _ = io.Copy(os.Stdout, response.Body)
			} else {
				fmt.Println("Response Header content type:", response.Header.Get("Content-Type"))
				downloadFile(response, fileName)
				fmt.Println(response.Body)
			}
		} else {
			fmt.Println("Post success")
		}
	}
}

func downloadFile(response *http.Response, fileName string) {
	// Download the file
	fmt.Println("Response status:", response.Status)
	fmt.Println("Response body:")
	//create file
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error create file:", err)
	}
	defer file.Close()
}
