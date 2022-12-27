// build a go server to serve for get and post requests from worker
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	handleRequests()
}

// Handle get and post requests
func handleRequests() {
	http.HandleFunc("/", homePage)
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
func homePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		fmt.Fprintf(w, "Welcome to the File storage server!")
		fmt.Println("Endpoint Hit: homePage")
	}
	if (r.URL.Path == "/root" || r.URL.Path == "/root/" || r.URL.Path == "/root/tmp" || r.URL.Path == "/root/tmp/") && r.Method == "GET" {
		file_names, err := ioutil.ReadDir("." + r.URL.Path)
		if err != nil {
			log.Fatal(err)
		}
		for _, file := range file_names {
			fmt.Fprintf(w, file.Name()+" ")
		}
		fmt.Println("Endpoint Hit: getFiles")
	} else {
		getFile(w, r)
	}

}
func getFile(w http.ResponseWriter, r *http.Request) {
	// return the file content
	if r.Method == "GET" {
		file_name := r.URL.Path
		file, err := os.Open("." + file_name)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		file_content, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatal(err)
		}
		w.Write(file_content)
	}
	if r.Method == "POST" {
		// write file content to the file
		fmt.Println("Endpoint Hit: postFile")
		file_name := r.URL.Path
		// if file exist, delete it
		if _, err := os.Stat("." + file_name); err == nil {
			err := os.Remove("." + file_name)
			if err != nil {
				log.Fatal(err)
			}
		}
		file, err := os.Create("." + file_name)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		file_content, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		file.Write(file_content)
	}
}
