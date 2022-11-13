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

func main() {
	// user input server address
	//Todo: Add proxy support

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

		fmt.Println("proxy test:")
		proxy(conn, root, fileName)
		fmt.Println("proxt test end")

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

const HttpProxy = "http://127.0.0.1:8081"

func proxy(conn *net.TCPConn, root string, fileName string) {

	proxy := func(_ *http.Request) (*url.URL, error) {
		return url.Parse(HttpProxy)
	}
	httpTransport := &http.Transport{Proxy: proxy}

	httpClient := &http.Client{Transport: httpTransport}
	host_addr := conn.RemoteAddr().String()

	url := "http://" + host_addr + root + "/" + fileName
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("proxy request Error:", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Println("proxy response Error:", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("proxy read resp body Error:", err)
	}
	fmt.Println(string(body))
	fmt.Println("Response Header content type:", resp.Header.Get("Content-Type"))

	//create file
	// check fileName is a file

	if strings.Contains(fileName, ".") {
		file, err := os.Create(fileName)
		if err != nil {
			panic(err)
		}
		defer func() { _ = file.Close() }()
		//write body to file
		file.Write(body)
	}

	//TODO: check why use func download get empty file
	//downloadFile(resp, fileName)
	fmt.Println("proxy download file over")

}

func sender(conn *net.TCPConn, method string, root string, fileName string) {
	host_addr := conn.RemoteAddr().String()

	url := "http://" + host_addr + root + "/" + fileName

	// Create a new request
	var request *http.Request
	if method == "GET" {
		request, _ = http.NewRequest(method, url, nil)
		// Todo: Use proxy

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

		//write file content to bytes
		bytes, err := ioutil.ReadAll(file)
		//fmt.Println("Bytes:", bytes)
		// read bytes into io.Reader
		reader := strings.NewReader(string(bytes))
		//fmt.Println("reader/request.body", reader)
		defer file.Close()

		file, err = os.Open(path)
		if err != nil {
			fmt.Println("Can not open file: ", err)
			return
		}

		// get the file content type for change the request header content type
		contentType, err := getFileContentType(file)

		if err != nil {
			fmt.Println("Get file content type error!")
		}

		fileEnding, _ := checkFileEnding(url)
		// if it is a css file change the content type(because default is text/plain)
		if fileEnding == "css" {
			contentType = "text/css; charset=utf-8"
		}
		defer file.Close()

		request, _ = http.NewRequest(method, url, reader)

		// add request header content type
		request.Header.Set("Content-Type", contentType)
		//fmt.Println("request header content type:", request.Header.Get("Content-Type"))

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
	file.Write(bytes)
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
