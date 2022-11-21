package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func TestConn(conn *net.TCPConn) bool {
	// Read ACK or WAIT from server
	reader := bufio.NewReader(conn)
	cnt := 0
	for {
		ack, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading ACK from server:", err)
			return false
		}
		ack = strings.TrimSpace(ack)
		if ack == "ACK" {
			fmt.Println("ACK received from server")
			return true
		} else {
			fmt.Println("Server is busy, waiting...")
			cnt++
		}
		if cnt == 3 {
			return false
		}
	}
}

func SendTCPConnACK(conn net.Conn) {
	// Send ACK to client
	_, err := conn.Write([]byte("ACK\n"))
	if err != nil {
		fmt.Println("Error sending ACK to client:", err)
	}
}

func SendTCPConnWAIT(conn net.Conn) {
	// Send WAIT to client
	_, err := conn.Write([]byte("WAIT\n"))
	if err != nil {
		fmt.Println("Error sending WAIT to client:", err)
	}
}

// Get address and port number from command line
func GetAddr() string {
	args := os.Args
	if len(args) != 2 {
		fmt.Println("Arguments length error! Using default address: localhost:8081")
		return "localhost:8081"
	}

	// Address should be like "ip:portnumber" or "portnumber"
	addr_list := strings.Split(args[1], ":")

	if len(addr_list) == 1 {
		// Check if the port number is valid
		port, err := strconv.Atoi(strings.TrimSpace(addr_list[0]))
		if err != nil {
			fmt.Println("Port number format error!, port is ", port)
			return "-1"
		}
		if port < 0 || port > 65535 {
			fmt.Println("Port number range error!")
			return "-1"
		}
		return "localhost:" + addr_list[0]
	} else if len(addr_list) == 2 {

		// Check if the address is valid
		if len(addr_list) != 2 {
			fmt.Println("Address format error!")
			return "-1"
		}

		// Check if the ip address is valid
		if addr_list[0] != "localhost" {
			ip := net.ParseIP(addr_list[0])
			if ip == nil {
				fmt.Println("IP address format error!")
				return "-1"
			}
		} // ip address is "localhost"

		// Check if the port number is valid
		port, err := strconv.Atoi(strings.TrimSpace(addr_list[1]))
		if err != nil {
			fmt.Println("Port number format error!, port is ", port)
			return "-1"
		}
		if port < 0 || port > 65535 {
			fmt.Println("Port number range error!")
			return "-1"
		}
		return args[1]
	} else {
		fmt.Println("Address format error! Using default address: localhost:8081")
		return "localhost:8081"
	}
}

func FirstRequestHandler(client net.Conn) net.Conn {
	var serverConn net.Conn
	if client == nil {
		return serverConn
	}
	var buffer [1024]byte
	client_request_msg, err := client.Read(buffer[:])
	if err != nil {
		if err == io.EOF {
			defer client.Close()
			fmt.Println("*******************************************************")
			fmt.Println("-------------------- Proxy over here ------------------")
			fmt.Println("*******************************************************")
		} else {
			fmt.Println("Error reading client buffer:", err.Error())
		}
		return serverConn
	}

	//Construct client request
	client_request, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(buffer[:client_request_msg])))
	if err != nil {
		fmt.Println("Error reading client request:", err.Error())
		return serverConn
	}

	//Construct client_request's Response
	client_request.Response = new(http.Response)

	// If method is not GET, return 501
	if client_request.Method != "GET" {
		client_request.Response.StatusCode = http.StatusNotImplemented
		client_request.Response.Write(client)
		return serverConn
	}

	//Construct server connection
	serverConn, err = net.Dial("tcp", client_request.Host)
	if err != nil {
		// Return internal server error
		client_request.Response.StatusCode = http.StatusInternalServerError
		client_request.Response.Write(client)
		fmt.Println("Error dialing server:", err.Error())
		return serverConn
	}
	defer serverConn.Close()

	// Construct proxy to server request
	src_url := client_request.URL
	fmt.Println("src_url: ", src_url)
	server_request, err := http.NewRequest("GET", src_url.String(), nil)
	if err != nil {
		// Return internal server error
		client_request.Response.StatusCode = http.StatusInternalServerError
		client_request.Response.Write(client)
		fmt.Println("Error constructing server request:", err.Error())
		return serverConn
	}

	// Copy client request header to server request
	for k, v := range client_request.Header {
		server_request.Header[k] = v
	}

	// Send server request
	err = server_request.Write(serverConn)
	if err != nil {
		// Return internal server error
		client_request.Response.StatusCode = http.StatusInternalServerError
		client_request.Response.Write(client)
		fmt.Println("Error writing server request:", err.Error())
		return serverConn
	}

	// Read server response
	server_response, err := http.ReadResponse(bufio.NewReader(serverConn), server_request)
	if err != nil {
		// Return internal server error
		client_request.Response.StatusCode = http.StatusInternalServerError
		client_request.Response.Write(client)
		fmt.Println("Error reading server response:", err.Error())
		return serverConn
	}
	defer server_response.Body.Close()

	// Copy server response attributes to client response
	client_request.Response.StatusCode = server_response.StatusCode
	client_request.Response.ContentLength = server_response.ContentLength
	client_request.Response.Header = server_response.Header
	client_request.Response.Body = server_response.Body

	// Send client response
	err = client_request.Response.Write(client)
	if err != nil {
		fmt.Println("Error writing client response:", err.Error())
		return serverConn
	}
	defer client_request.Response.Body.Close()
	return serverConn
}
