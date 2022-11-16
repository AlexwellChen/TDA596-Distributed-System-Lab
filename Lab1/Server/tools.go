package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func GetFileContentType(out *os.File) (string, error) {
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

func CheckFileEnding(url string) (string, bool) {
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

func DownloadFile(request *http.Request, fileName string) error {
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

	// Creat buffer to store the file

	bytes, err := io.ReadAll(request.Body)
	if err != nil {
		fmt.Println("Error reading file:", err)
	}
	// Write the file to disk
	_, err = file.Write(bytes)
	fmt.Println("POST download Bytes length:", len(bytes))
	fmt.Println("request content-length:", request.ContentLength)
	return err
}

// Get port number from command line
func GetPort() int {
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
