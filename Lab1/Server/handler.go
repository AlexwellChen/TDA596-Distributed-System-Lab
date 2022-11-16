package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Handler for GET request
func GetHandler(r *http.Request) (StatusCode int) {
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
		response.Body = io.NopCloser(strings.NewReader(resp_not_found))
		return response.StatusCode
	}

	// Check if file or directory could be read
	file, err := os.Open(url)
	if err != nil {
		fmt.Println("Open file or directory error: ", err)
		// Return resource not found
		response.StatusCode = http.StatusNotFound
		response.ContentLength = int64(len(resp_not_found))
		response.Body = io.NopCloser(strings.NewReader(resp_not_found))
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
			response.Body = io.NopCloser(strings.NewReader(resp_internal_err))
			return response.StatusCode
		}

		// file_info to string
		var file_info_str string
		for _, file := range file_info {
			file_info_str += file.Name() + " "
		}
		response.StatusCode = http.StatusOK
		response.ContentLength = int64(len(file_info_str))
		response.Body = io.NopCloser(strings.NewReader(file_info_str))
	} else {
		// if it is a file
		// get the file content type for transmitting them to client
		contentType, err := GetFileContentType(file)

		if err != nil {
			fmt.Println("Get file content type error!")
		}

		//check if the file is we need file
		fileEnding, valid := CheckFileEnding(url)
		// if it is a css file change the content type(because default is text/plain)
		if fileEnding == "css" {
			contentType = "text/css; charset=utf-8"
		}
		if valid {
			// TODO: check why file was closed before, has to reopen otherwise will get is empty
			file, _ = os.Open(url)
			content, err := io.ReadAll(file)
			if err != nil {
				fmt.Println("Read file error!")
				// Return internal server error
				response.StatusCode = http.StatusInternalServerError
				response.ContentLength = int64(len(resp_internal_err))
				response.Body = io.NopCloser(strings.NewReader(resp_internal_err))
				return response.StatusCode
			}
			if response.Header == nil {
				response.Header = make(http.Header)
			}
			response.Header.Add("Content-Type", contentType)
			response.StatusCode = http.StatusOK
			response.ContentLength = int64(len(content))
			fmt.Println("Content length: ", response.ContentLength)
			response.Body = io.NopCloser(strings.NewReader(string(content)))
		} else {
			// Return internal server error
			response.StatusCode = http.StatusBadRequest
			response.ContentLength = int64(len(resp_bad_request))
			response.Body = io.NopCloser(strings.NewReader(resp_bad_request))
		}
	}
	return response.StatusCode
}

// Handler for POST request
func PostHandler(r *http.Request) (StatusCode int) {
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
	_, valid := CheckFileEnding(url)

	if valid {
		pwd, _ := os.Getwd()
		url = pwd + url
		// Check if file exists
		err := DownloadFile(r, url)

		// reqBody, err := io.ReadAll(r.Body)
		// print thr length of request body
		// fmt.Println("Request body length: ", len(reqBody))
		// fmt.Println("Request body: ", string(reqBody))
		if err != nil {
			fmt.Println("Download file error: ", err)
			response.StatusCode = http.StatusInternalServerError
			return response.StatusCode
		}

		response.StatusCode = http.StatusOK
		response.ContentLength = int64(len("OK"))
		response.Body = io.NopCloser(strings.NewReader("OK"))
	} else {
		response.StatusCode = http.StatusBadRequest
		response.ContentLength = int64(len("Bad request"))
		response.Body = io.NopCloser(strings.NewReader("Bad request"))
	}
	return response.StatusCode
}

// Handler for other request, return status code
func UnsupportedMethodHandler(r *http.Request) (StatusCode int) {
	response := r.Response
	response.StatusCode = http.StatusNotImplemented //501
	response.Body = io.NopCloser(strings.NewReader("Method not allowed"))
	fmt.Println("Unsupported method!")
	return response.StatusCode
}
