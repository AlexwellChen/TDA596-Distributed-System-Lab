### Lab 1 Server/Client
run `go get -u golang.org/x/sync/semaphore` to get semaphore package

#### Server
**Start Server**
`cd Server`
Run server with `go run main.go [ip address:]<port number>` 

eg. `go run main.go 8080` or `go run main.go localhost:8080`

#### Client
**Start Client**
`cd Client`
Run client with `go run main.go`

#### Proxy
**Start Proxy**
`cd Proxy`
Run proxy with `go run main.go <port number>`

TODOs: 
add content-type in request header of POST method
add client exit notification when server stopped connection or exits
add other file type statuscode for bad request
fix posting issue:
    - Posting a jpg file, the request body on server side is different from client side (**DONE**)
    - client side is clear, checking server request body



