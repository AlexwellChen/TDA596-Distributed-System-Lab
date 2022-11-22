### Lab 1 Server/Client
run `go get -u github.com/zh-five/golimit` to get golimit package  


**Start Server**
`cd Server`  
Run server with `go run *.go [ip address:]<port number>`  
Build server with `go build -o http-server` and run with `./http-server [ip address:]<port number>`  

eg. `go run *.go 8080` or `go run *.go localhost:8080`  

**Start Client**
`cd Client`  
Run client with `go run *.go`  
Build client with `go build -o http-client` and run with `./http-client`  


**Start Proxy**
`cd Proxy`  
Run proxy with `go run main.go <port number>`  
Build proxy with `go build -o http-proxy` and run with `./http-proxy <port number>`  



