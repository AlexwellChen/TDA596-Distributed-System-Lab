package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
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

// Handler for GET request
func getHandler(r *http.Request) (StatusCode int) {
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
		return response.StatusCode
	}

	// Check if file or directory could be read
	file, err := os.Open(url)
	if err != nil {
		fmt.Println("Open file or directory error: ", err)
		// Return resource not found
		response.StatusCode = http.StatusNotFound
		response.ContentLength = int64(len(resp_not_found))
		response.Body = ioutil.NopCloser(strings.NewReader(resp_not_found))
		return response.StatusCode
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
			return response.StatusCode
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
		// if it is a file
		// get the file content type for transmitting them to client
		contentType, err := getFileContentType(file)

		if err != nil {
			fmt.Println("Get file content type error!")
		}

		//check if the file is we need file
		fileEnding, valid := checkFileEnding(url)
		// if it is a css file change the content type(because default is text/plain)
		if fileEnding == "css" {
			contentType = "text/css; charset=utf-8"
		}
		if valid {
			// TODO: check why file was closed before, has to reopen otherwise will get is empty
			file, _ = os.Open(url)
			content, err := ioutil.ReadAll(file)
			if err != nil {
				fmt.Println("Read file error!")
				// Return internal server error
				response.StatusCode = http.StatusInternalServerError
				response.ContentLength = int64(len(resp_internal_err))
				response.Body = ioutil.NopCloser(strings.NewReader(resp_internal_err))
				return
			}
			if response.Header == nil {
				response.Header = make(http.Header)
			}
			response.Header.Add("Content-Type", contentType)
			response.StatusCode = http.StatusOK
			response.ContentLength = int64(len(content))
			fmt.Println("Content length: ", response.ContentLength)
			response.Body = ioutil.NopCloser(strings.NewReader(string(content)))
		} else {
			// Return internal server error
			response.StatusCode = http.StatusBadRequest
			response.ContentLength = int64(len(resp_bad_request))
			response.Body = ioutil.NopCloser(strings.NewReader(resp_bad_request))
		}
	}
	return response.StatusCode
}

// Handler for POST request
func postHandler(r *http.Request) (StatusCode int) {
	// TODO: fix runtime error: invalid memory address or nil pointer dereference
	fmt.Println("Invoke POST Handler")

	response := r.Response
	//test Content-Type
	fmt.Println("Request header content type: ", r.Header.Get("Content-Type"))

	url := r.URL.Path
	fmt.Println("URL: ", url)

	bodylength := r.ContentLength
	fmt.Println("Contentent length: ", bodylength)
	// Check file type
	// TODO: css file type to test
	_, valid := checkFileEnding(url)

	if valid {
		pwd, _ := os.Getwd()
		url = pwd + url
		// Check if file exists
		// err := downloadFile(r, url)

		reqBody, err := ioutil.ReadAll(r.Body)
		// print thr length of request body
		fmt.Println("Request body length: ", len(reqBody))
		if err != nil {
			fmt.Println("Download file error: ", err)
			response.StatusCode = http.StatusInternalServerError
			return
		}

		response.StatusCode = http.StatusOK
		response.ContentLength = int64(len("OK"))
		response.Body = ioutil.NopCloser(strings.NewReader("OK"))
	} else {
		response.StatusCode = http.StatusBadRequest
		response.ContentLength = int64(len("Bad request"))
		response.Body = ioutil.NopCloser(strings.NewReader("Bad request"))
	}
	return response.StatusCode
}

// Handler for other request, return status code
func unsupportedMethodHandler(r *http.Request) (StatusCode int) {
	response := r.Response
	response.StatusCode = http.StatusNotImplemented //501
	response.Body = ioutil.NopCloser(strings.NewReader("Method not allowed"))
	fmt.Println("Unsupported method!")
	return response.StatusCode
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
			if err == io.EOF {
				fmt.Println("Connection closed!")
			} else {
				fmt.Println("connection err!:", err)
			}
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
		var respCode int
		if request.Method == "GET" {
			respCode = getHandler(request)
		} else if request.Method == "POST" {
			respCode = postHandler(request)
		} else {
			respCode = unsupportedMethodHandler(request)
		}
		request.Response.Write(conn)
		fmt.Println("Send response", respCode, "successfully!")
		defer request.Response.Body.Close()
		fmt.Println("--------------------------------------------------")
	}
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

func downloadFile(request *http.Request, fileName string) error {
	// Download the file

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
	bytes, err := ioutil.ReadAll(request.Body)
	_, err = file.Write(bytes)
	fmt.Println("POST download Bytes length:", len(bytes))
	fmt.Println("request content-length:", request.ContentLength)
	return err
}
