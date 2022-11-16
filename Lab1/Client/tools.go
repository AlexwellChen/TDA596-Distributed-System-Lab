package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

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

func DownloadFile(response *http.Response, fileName string) {
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
	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
	}
	file.Write(bytes)
	// fmt.Println("Bytes:", bytes)
}
