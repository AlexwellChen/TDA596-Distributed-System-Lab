package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sync/semaphore"
)

const (
	Limit  = 10 // Upper limit of concurrent connections
	Weight = 1  // Weight of each connection
)

// global variable semaphore
var s = semaphore.NewWeighted(Limit)

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
	resp_not_found := "Resource not found"
	if err != nil {
		fmt.Println("Status error: ", err)
		response.StatusCode = http.StatusNotFound
		response.ContentLength = int64(len(resp_not_found))
		response.Body = ioutil.NopCloser(strings.NewReader(resp_not_found))
		return
	}

	// Check if file or directory could be read
	file, err := os.Open(url)
	if err != nil {
		fmt.Println("Open file or directory error: ", err)
		// Return resource not found
		response.StatusCode = http.StatusNotFound
		response.ContentLength = int64(len(resp_not_found))
		response.Body = ioutil.NopCloser(strings.NewReader(resp_not_found))
		return
	}
	defer file.Close()

	// Check if it is a directory
	resp_internal_err := "Internal server error"
	resp_bad_request := "Bad request"
	if s.IsDir() {
		// a directory
		file_info, err := file.Readdir(-1)
		if err != nil {
			fmt.Println("Read file error!")
			// Return internal server error
			response.StatusCode = http.StatusInternalServerError
			response.ContentLength = int64(len(resp_internal_err))
			response.Body = ioutil.NopCloser(strings.NewReader(resp_internal_err))
			return
		}

		// file_info to string
		var file_info_str string
		for _, file := range file_info {
			file_info_str += file.Name() + " "
		}
		response.StatusCode = http.StatusOK
		response.ContentLength = int64(len(file_info_str))
		response.Body = ioutil.NopCloser(strings.NewReader(file_info_str))
	} else {
		// a file
		//create a array to store url which is split by "/"
		arrUrl := strings.Split(url, ".")
		//get the last element of the array -> Ending of file name
		//(e.g. html, txt, gif, jpeg, jpg or css)
		fileNameEnding := arrUrl[len(arrUrl)-1]
		//fmt.Println("Ending of File name: ", fileNameEnding)
		// TODO: use function to get the file content type for transmitting them to client
		contentType, err := getFileContentType(file)

		if err != nil {
			fmt.Println("Get file content type error!")
		}
		fmt.Println("Content type and file name ending: ", contentType, fileNameEnding)

		//check if the file is we need file
		if fileNameEnding == "html" || fileNameEnding == "txt" || fileNameEnding == "css" || fileNameEnding == "gif" || fileNameEnding == "jpeg" || fileNameEnding == "jpg" {
			bytes, err := ioutil.ReadAll(file)
			if response.Header == nil {
				response.Header = make(map[string][]string)
			}
			response.Header.Add("Content-Type", contentType)
			fmt.Println(contentType)
			if err != nil {
				fmt.Println("Read file error!")
				// Return internal server error
				response.StatusCode = http.StatusInternalServerError
				response.ContentLength = int64(len(resp_internal_err))
				response.Body = ioutil.NopCloser(strings.NewReader(resp_internal_err))
				return
			}
			response.StatusCode = http.StatusOK
			response.ContentLength = int64(len(bytes))
			response.Body = ioutil.NopCloser(strings.NewReader(string(bytes)))
		} else {
			// Return internal server error
			response.StatusCode = http.StatusBadRequest
			response.ContentLength = int64(len(resp_bad_request))
			response.Body = ioutil.NopCloser(strings.NewReader(resp_bad_request))
			return
		}
		/*		bytes, err := ioutil.ReadAll(file)
				if err != nil {
					fmt.Println("Read file error!")
					// Return internal server error
					response.StatusCode = http.StatusInternalServerError
					response.ContentLength = int64(len(resp_internal_err))
					response.Body = ioutil.NopCloser(strings.NewReader(resp_internal_err))
					return
				}
				response.StatusCode = http.StatusOK
				response.ContentLength = int64(len(bytes))
				response.Body = ioutil.NopCloser(strings.NewReader(string(bytes)))*/
	}

}

func postHandler(r *http.Request) {
	fmt.Println("Invoke POST Handler")
	response := r.Response
	url := r.URL.Path
	fmt.Println("URL: ", url)
	pwd, _ := os.Getwd()
	url = pwd + url
	content := r.Body
	fmt.Println("Content: ", content)
	//TODO: save file to local root directory
	// TODO: check file type
	response.StatusCode = http.StatusOK

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
	// defer listener.Close()
	fmt.Println("Listening on " + address)
	//Todo: Add concurrency control here, maxmum 10 connections
	ctx := context.TODO()
	// TODO returns a non-nil, empty Context.
	// Code should use context.TODO when it's unclear which Context to use or it is not yet available
	// (because the surrounding function has not yet been extended to accept a Context parameter)
	for {
		//acquire semaphore

		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept err!: ", err)
		} else {
			err = s.Acquire(ctx, Weight)
			if err != nil {
				fmt.Println("Semaphore full!")
			} else {
				go handleConnection(conn, root)
			}
		}

	}
}

func handleConnection(conn net.Conn, root string) {
	//Create an empty buffer
	buffer := make([]byte, 1024)

	// read from connection
	for {
		msg, err := conn.Read(buffer)
		if err != nil {
			// handle error
			fmt.Println("connection err!:", err)
			conn.Close()
			//release semaphore
			s.Release(Weight)
			return
		}
		// print message
		fmt.Println("Request from ", conn.RemoteAddr().String())

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
}

func getFileContentType(out *os.File) (string, error) {
	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	n, err := out.Read(buffer)
	fmt.Println("Read file content: ", buffer)
	if err != nil {
		return "", err
	}
	buffer = buffer[:n]
	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)
	return contentType, nil
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
