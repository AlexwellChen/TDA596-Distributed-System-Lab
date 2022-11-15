package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const HttpProxy = "http://127.0.0.1:8081"

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
	if proxyNeedYes {
		fmt.Println("Proxy connection...")
	} else {
		fmt.Println("No proxy connection...")
	}

	fmt.Println("Please enter <server address>:<Port number>, e.g. 127.0.0.1:8080")
	//reader := bufio.NewReader(os.Stdin)
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
			proxy(conn, method, root, fileName)
			fmt.Println("proxt test end")
			//}
		} else {
			sender(conn, method, root, fileName)
		}

		//sender(conn, method, root, fileName)
		fmt.Println("--------------------------------------------------")
	}
}

func proxy(conn *net.TCPConn, method string, root string, fileName string) {

	proxy := func(_ *http.Request) (*url.URL, error) {
		return url.Parse(HttpProxy)
	}
	httpTransport := &http.Transport{Proxy: proxy}

	httpClient := &http.Client{Transport: httpTransport}
	host_addr := conn.RemoteAddr().String()

	url := "http://" + host_addr + root + "/" + fileName

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Println("proxy request Error:", err)
	}

	resp, err := httpClient.Do(req)
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
			downloadFile(resp, fileName)
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
		contentType, err := getFileContentType(file)
		if err != nil {
			fmt.Println("Get file content type error!")
		}

		//write file content to bytes
		file, _ = os.Open(path)
		bytes, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Println("Read file error!")
		}
		//fmt.Println("Bytes:", bytes)
		// read bytes into io.Reader
		reader := strings.NewReader(string(bytes))
		// fmt.Println("reader/request.body", reader)

		fileEnding, _ := checkFileEnding(url)
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
		fmt.Println("contentType:", request.Header.Get("Content-Type"))
		// Content-Length is set automatically by http.NewRequest
		fmt.Println("POST upload bytes length:", request.ContentLength)

	} else {
		request, _ = http.NewRequest(method, url, nil) //unspoorted method also need to send request
		fmt.Println("Invalid request method!")
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
				_, _ = io.Copy(os.Stdout, response.Body)
			} else {
				fmt.Println("Response Header content type:", response.Header.Get("Content-Type"))
				downloadFile(response, fileName)
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
		//TODO: if file(1) exists, create a new file with the file(2)?
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
	if err != nil {
		fmt.Println("Error reading response body:", err)
	}
	file.Write(bytes)
	// fmt.Println("Bytes:", bytes)
}

func getFileContentType(out *os.File) (string, error) {
	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	n, err := out.Read(buffer)
	if err != nil {
		return "", err
	}
	buffer = buffer[:n]
	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)
	return contentType, nil
}

func checkFileEnding(url string) (string, bool) {
	// Check file type
	//create a array to store url which is split by "/"
	arrUrl := strings.Split(url, ".")
	//get the last element of the array -> Ending of file name
	//(e.g. html, txt, gif, jpeg, jpg or css)
	fileNameEnding := arrUrl[len(arrUrl)-1]
	if fileNameEnding == "html" || fileNameEnding == "txt" || fileNameEnding == "gif" || fileNameEnding == "jpeg" || fileNameEnding == "jpg" || fileNameEnding == "css" {
		return fileNameEnding, true
	} else {
		return fileNameEnding, false
	}
}
