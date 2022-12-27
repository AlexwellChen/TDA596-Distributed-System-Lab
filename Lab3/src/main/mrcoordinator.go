package main

//
// start the coordinator process, which is implemented
// in ../mr/coordinator.go
//
// go run mrcoordinator.go pg*.txt
//
// Please do not change this file.
//

import (
	"fmt"
	"os"
	"time"

	"6.824/mr"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: mrcoordinator inputfiles...\n")
		os.Exit(1)
	}

	fmt.Println("mrcoordinator: starting coordinator process")
	fmt.Println("mrcoordinator: input files are", os.Args[1:len(os.Args)-1])
	m := mr.MakeCoordinator(os.Args[1:len(os.Args)-1], 10, os.Args[len(os.Args)-1])
	for m.Done() == false {
		time.Sleep(time.Second)
	}

	time.Sleep(time.Second)
}
