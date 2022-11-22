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
	// ask for proxy support
	fmt.Println("Please enter if you want to use proxy (y/n):")
	reader := bufio.NewReader(os.Stdin)
	proxyNeed, _ := reader.ReadString('\n')
	proxyNeed = strings.TrimSpace(proxyNeed)
	//case insensitive
	proxyNeed = strings.ToUpper(strings.TrimSpace(proxyNeed))
	proxyNeedYes := proxyNeed == "Y" || proxyNeed == "YES"
	var HttpProxy string
	if proxyNeedYes {
		fmt.Println("If you want use default address, just press Enter")
		fmt.Println("Please enter the proxy address:")
		HttpProxy = SetProxyAddr()
		if HttpProxy == "-1" {
			fmt.Println("Using default proxy address: localhost:8081")
			HttpProxy = "localhost:8081"
		}
		fmt.Println("Proxy address:", HttpProxy, "is connecting")
	} else {
		fmt.Println("No proxy connection...")
	}

	fmt.Println("--------------------------------------")
	fmt.Println("Please enter [server address]:<Port number>, e.g. 127.0.0.1:8080 or 8080")
	fmt.Println("If you want to use default address, just press Enter")
	server := GetClientAddr()
	if server == "-1" {
		fmt.Println("Using default address: localhost:8080")
		server = "localhost:8080"
	}
	fmt.Println("--------------------------------------")
	server = strings.TrimSpace(server)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", server)

	if err != nil {
		fmt.Println("Fatal error: ", err)
		os.Exit(1)
	}

	// Connect to server or proxy
	if proxyNeedYes {
		// use proxy
		proxyTCPAddr, err := net.ResolveTCPAddr("tcp4", HttpProxy)
		if err != nil {
			fmt.Println("Fatal error: ", err)
			os.Exit(1)
		}
		tcpAddr = proxyTCPAddr
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		// if connection was refused, conn does not exist and gives a nil pointer error
		fmt.Println("Fatal error:", err)
		os.Exit(1)
	}
	// Send hostaddr to proxy
	if proxyNeedYes {
		_, err = conn.Write([]byte(server + "\n"))
		if err != nil {
			fmt.Println("Error sending hostaddr to proxy:", err)
		}
	}
	fmt.Println("connection success")
	for {
		// Todo: Add a UNIX style command line interface
		//repeat send request until user input "exit"
		//Ask user for input request resource and method?
		fmt.Println("Please enter request method, or enter exit to exit connection:") //GET POST
		method, _ := reader.ReadString('\n')
		//case insensitive
		method = strings.ToUpper(strings.TrimSpace(method))
		if method == "EXIT" {
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

		if proxyNeedYes {
			fmt.Println("proxy test:")
			proxy(conn, method, root, fileName, server)
			fmt.Println("proxt test end")
			//}
		} else {
			sender(conn, method, root, fileName)
		}
		fmt.Println("--------------------------------------------------")
	}
}

func proxy(conn *net.TCPConn, method string, root string, fileName string, hostAddr string) {
	// send request to proxy
	host_addr := hostAddr

	url := "http://" + host_addr + root + "/" + fileName

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Println("proxy request Error:", err)
	}

	// resp, err := httpClient.Do(req)
	err = req.Write(conn)
	if err != nil {
		fmt.Println("proxy request Error:", err)
	}
	// Read response from server
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		fmt.Println("proxy response Error:", err)
	}
	defer resp.Body.Close()

	// create file
	// check fileName is a file or a directory
	if resp.StatusCode == 200 {
		if fileName == "" {
			fmt.Println("The files in the directory have been listed below:")
			_, _ = io.Copy(os.Stdout, resp.Body)
		} else {
			fmt.Println("Response Header content type:", resp.Header.Get("Content-Type"))
			DownloadFile(resp, fileName)
		}
	} else {
		fmt.Println("Proxy only resonse to GET method\n The StatusCode is:", resp.StatusCode)
		fmt.Println("501 Not Implemented")
	}

	fmt.Println("Proxy success")

}

func sender(conn *net.TCPConn, method string, root string, fileName string) {
	host_addr := conn.RemoteAddr().String()

	url := "http://" + host_addr + root + "/" + fileName

	// Create a new request
	var request *http.Request

	fmt.Println("***************** Sender *****************")
	if method == "GET" {
		request, _ = http.NewRequest(method, url, nil)
		fmt.Println("GET url:", url)
	} else if method == "POST" {
		//TODO: post error: use post then get -> error  and  post many times -> error
		//TODO: post jpg file, can create file but isn't show content

		pwd, _ := os.Getwd()
		path := pwd + root + "/" + fileName
		fmt.Println("POST local path:", path)

		file, err := os.Open(path)
		if err != nil {
			fmt.Println("Can not open file: ", err)
			return
		}
		defer file.Close()
		// get the file content type for change the request header content type
		contentType, err := GetFileContentType(file)
		if err != nil {
			fmt.Println("Get file content type error!")
		}

		//write file content to bytes
		file, _ = os.Open(path)
		bytes, err := io.ReadAll(file)
		if err != nil {
			fmt.Println("Read file error!")
		}
		//fmt.Println("Bytes:", bytes)
		// read bytes into io.Reader
		reader := strings.NewReader(string(bytes))
		// fmt.Println("reader/request.body", reader)

		fileEnding, _ := CheckFileEnding(url)
		// if it is a css file change the content type(because default is text/plain)
		if fileEnding == "css" {
			contentType = "text/css; charset=utf-8"
		}

		request, err = http.NewRequest(method, url, reader)
		if err != nil {
			fmt.Println("New request error:", err)
		}
		defer request.Body.Close()
		// add request header content type

		request.Header.Add("Content-Type", contentType)
		request.Close = true
		fmt.Println("contentType:", request.Header.Get("Content-Type"))
		// Content-Length is set automatically by http.NewRequest
		fmt.Println("POST upload bytes length:", request.ContentLength)

	} else {
		if method == "" {
			fmt.Println("Please enter request method!")
			return
		} else {
			request, _ = http.NewRequest(method, url, nil) //unspoorted method also need to send request
			// Default is GET
			fmt.Println("Invalid request method: ", request.Method)
		}
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
	// go automatically breaks after first match
	case http.StatusInternalServerError:
		fmt.Println("500 Internal Server Error")
	case http.StatusNotImplemented:
		fmt.Println("501 Not Implemented")
	case http.StatusBadGateway:
		fmt.Println("502 Bad Gateway")
	case http.StatusBadRequest:
		fmt.Println("400 Bad Request")
	case http.StatusNotFound:
		fmt.Println("404 Not Found")
	case http.StatusOK:
		fmt.Println("200 OK")
		if method == "GET" {
			if fileName == "" {
				fmt.Println("The files in the directory have been listed below:")
				// Print response body with println
				bodyString, _ := io.ReadAll(response.Body)
				fmt.Println(string(bodyString))
				// _, _ = io.Copy(os.Stdout, response.Body)
			} else {
				fmt.Println("Response Header content type:", response.Header.Get("Content-Type"))
				DownloadFile(response, fileName)
			}
		} else if method == "POST" {
			fmt.Println("Post success")
		} else {
			fmt.Println("Invalid request method!")
		}
	default:
		fmt.Println("Invalid request method!")
	}
}
