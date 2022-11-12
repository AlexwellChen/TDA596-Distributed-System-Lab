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
	server, _ := reader.ReadString('\n')
	// server := "localhost:8080"
	// fmt.Println(strings.TrimSpace(server))
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
		fmt.Println("Please enter request method:")
		method, _ := reader.ReadString('\n')
		method = strings.TrimSpace(method)
		fmt.Println("Please enter request resource:")
		resource, _ := reader.ReadString('\n')
		resource = strings.TrimSpace(resource)
		sender(conn, method, resource)
		fmt.Println("send over")
	}

}

func sender(conn *net.TCPConn, method string, resource string) {
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

	// Handle response body
	// Todo: 区分请求内容，如果是路径，需要列出目录下的文件；如果是文件，需要读取文件内容并保存到本地
	//需要区分文件后缀类型..需要修改
	if response.StatusCode == 200 {
		fmt.Println("Response status:", response.Status)
		fmt.Println("Response body:")
		//create file
		file, err := os.Create("Hello world.txt")
		if err != nil {
			fmt.Println("Error create file:", err)
		}
		defer func() { _ = file.Close() }()

		//Read the content and write into file
		copyFile, err := io.Copy(file, response.Body)
		fmt.Println(copyFile)
	} else if response.StatusCode == 404 {
		fmt.Println("Response status:", response.Status)
		fmt.Println("Bad Request")
	} else if response.StatusCode == 501 {
		fmt.Println("Response status:", response.Status)
		fmt.Println("Not Implemented")
	}

	/*	body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Error reading body:", err)
		}
		fmt.Println("Response body:\n", string(body))*/
	/*	list := url
		fmt.Println(list)
		filePath := list[1]
		fmt.Println("filepath:", filePath)
		info, err := os.Stat(string(filePath))
		if err != nil {
			fmt.Println("Error reading file:", err)
		}
		filename := info.Name()
		fmt.Println("filename:", filename)*/
}
