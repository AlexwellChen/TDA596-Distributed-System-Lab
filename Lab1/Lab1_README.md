### Lab 1 Server/Client
run `go get -u golang.org/x/sync/semaphore` to get semaphore package

#### Server
**Start Server**
Run server with `go run main.go <port number>`

#### Client
**Start Client**
Run client with `go run main.go`

TODOs: 
add content-type in request header of POST method
add client exit notification when server stopped connection or exits
add other file type statuscode for bad request
fix posting issue:
    - Posting a jpg file, the request body on server side is different from client side
    - client side is clear, checking server request body


 

