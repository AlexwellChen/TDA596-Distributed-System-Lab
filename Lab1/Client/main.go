package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
)

func main() {
	// user input server address
	fmt.Println("Please enter <server address>:<Port number>, e.g. 127.0.0.1:8080")
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
		// Todo: Add a UNIX style command line interface
		//repeat send request until user input "exit"
		//Ask user for input request resource and method?
		fmt.Println("Please enter request method, or enter exit to exit connection:") //GET POST
		method, _ := reader.ReadString('\n')
		method = strings.TrimSpace(method)
		if method == "exit" {
			fmt.Println("Exiting connection...")
			conn.Close()
			break
		}
		fmt.Println("Please enter request resource root:") //root
		root, _ := reader.ReadString('\n')
		root = "/" + strings.TrimSpace(root)
		fmt.Println("Please enter request file:") // file name
		fileName, _ := reader.ReadString('\n')
		fileName = strings.TrimSpace(fileName)

		sender(conn, method, root, fileName)
		fmt.Println("--------------------------------------------------")
	}
	/*	method := "GET"
		file := "2.jpg"
		resource := "/root/" + file
		sender(conn, method, resource, file)
		//sender(conn, method, resource)
		fmt.Println("send over")*/

}

func sender(conn *net.TCPConn, method string, root string, fileName string) {
	host_addr := conn.RemoteAddr().String()

	url := "http://" + host_addr + root + "/" + fileName

	// Create a new request
	var request *http.Request
	if method == "GET" {
		request, _ = http.NewRequest(method, url, nil)
		fmt.Println("GET url:", url)
	} else if method == "POST" {
		pwd, _ := os.Getwd()
		path := pwd + root + fileName
		fmt.Println("POST local path:", path)
		file, err := os.Open(path)
		if err != nil {
			fmt.Println("Can not open file: ", err)
			return
		}
		bytes, err := ioutil.ReadAll(file)
		// read bytes into io.Reader
		// fmt.Println("Bytes:", bytes)
		reader := strings.NewReader(string(bytes))
		request, _ = http.NewRequest(method, url, reader)
		file.Close()
	} else {
		fmt.Println("Invalid request method!")
		return
	}
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
	case http.StatusNotFound:
		fmt.Println("404 Not Found")
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
			}
		} else {
			fmt.Println("Post success")
		}
	}
}

func downloadFile(response *http.Response, fileName string) {
	// Download the file
	fmt.Println("Response status:", response.Status)

	//check if file exists
	_, err := os.Stat(fileName)
	var file *os.File
	if err == nil {
		//create file
		fmt.Println("File already exists! Creating new file...")
		//TODO: os.Open needs additional parameters to overwrite the file
		// or use create filename.(1) to write to a new file
		file, err = os.Create(strings.Split(fileName, ".")[0] + "(1)." + strings.Split(fileName, ".")[1])
		if err != nil {
			fmt.Println("Error creating file:", err)
		}
	} else {
		fmt.Println("File does not exist, creating new file...")
		file, err = os.Create(fileName)
		if err != nil {
			fmt.Println("Error creating file:", err)
		}
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(response.Body)
	file.Write(bytes)
}
