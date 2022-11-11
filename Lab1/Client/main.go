package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
)

func main() {
	server := "localhost:8080"
	tcpAddr, err := net.ResolveTCPAddr("tcp4", server)

	if err != nil {
		fmt.Println(os.Stderr, "Fatal error: ", err)
		os.Exit(1)
	}

	// Connect to server
	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	if err != nil {
		fmt.Println(conn.RemoteAddr().String(), os.Stderr, "Fatal error:", err)
		os.Exit(1)
	}

	defer conn.Close()

	fmt.Println("connection success")
	sender(conn)
	fmt.Println("send over")

}

func sender(conn *net.TCPConn) {
	host_addr := conn.RemoteAddr().String()

	url := "http://" + host_addr + "/root"
	fmt.Println("url:", url)

	// Create a new request
	request, _ := http.NewRequest("GET", url, nil)
	err := request.Write(conn)
	if err != nil {
		fmt.Println(conn.RemoteAddr().String(), " Error: ", err)
		os.Exit(1)
	}

	// Read response from connection
	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, request)
	if err != nil {
		fmt.Println("Error reading response:", err)
	}
	defer response.Body.Close()

	// Handle response body
	// Todo: 区分请求内容，如果是路径，需要列出目录下的文件；如果是文件，需要读取文件内容并保存到本地
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading body:", err)
	}
	fmt.Println("Response body:\n", string(body))
}
