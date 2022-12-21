### Lab2 Chord 

### Usage

The Chord client will be a command-line utility which takes the following arguments:

1. -a <String> = The IP address that the Chord client will bind to, as well as advertise to other nodes. Represented as an ASCII string (e.g., 128.8.126.63). Must be specified.
2. -p <Number> = The port that the Chord client will bind to and listen on. Represented as a base-10 integer. Must be specified.
3. --ja <String> = The IP address of the machine running a Chord node. The Chord client will join this node’s ring. Represented as an ASCII string (e.g., 128.8.126.63). Must be specified if --jp is specified.
4. --jp <Number> = The port that an existing Chord node is bound to and listening on. The Chord client will join this node’s ring. Represented as a base-10 integer. Must be specified if --ja is specified.
5. --ts <Number> = The time in milliseconds between invocations of ‘stabilize’. Represented as a base-10 integer. Must be specified, with a value in the range of [1,60000].
6. --tff <Number> = The time in milliseconds between invocations of ‘fix fingers’. Represented as a base-10 integer. Must be specified, with a value in the range of [1,60000].
7. --tcp <Number> = The time in milliseconds between invocations of ‘check predecessor’.
   Represented as a base-10 integer. Must be specified, with a value in the range of [1,60000].
8. -r <Number> = The number of successors maintained by the Chord client. Represented as a base-10 integer. Must be specified, with a value in the range of [1,32].
9. -i <String> = The identifier (ID) assigned to the Chord client which will override the ID computed by the SHA1 sum of the client’s IP address and port number. Represented as a string of 40 characters matching [0-9a-fA-F]. Optional parameter.

Exp:

Start a chord at localhost:8000

`chord -a localhost -p 8000 --ts 30000 --tff 10000 --tcp 30000 -r 4`

Join a chord from localhost:8080 at localhost:8000

`chord -a localhost -p 8080 --ja localhost --jp 8000 --ts 30000 --tff 10000 --tcp 30000 -r 4`

### Comm between Node

We are using jsonrpc as comm method. Each remote method invoke shoud use *ChordCall* function.

Each RPC method should follow Golang RPC style and coding as following style.

```go
type MethodRPCReply struct {
	Variable var_type // Name starting with a uppercase letter
}

func (node *Node) method (args) retVar {
  // Local method, Name starting with a lowercase letter
  /*Start process*/
  return retVar
}

func (node *Node) MethodRPC(Request interface{}, reply *MethodRPCReply) error {
  // Remote method, Name starting with a uppercase letter and end with 'RPC'
  /*Start process*/
  return err
}
```

### Module Description

* main.go: 

  Responsible for the creation of the node and the start of the Chord service, as well as handling the user's command input and calling the corresponding methods.

* tools.go:

  Responsible for aiding in Chord ring creation, communication, and command line input processing.

* node.go:

  Responsible for defining the structure of the node and the local and remote RPC methods related to the node's own properties.

* routing.go

  Responsible for node and file lookup and routing functions on the chord.

* stabilizing.go

  Responsible for the stability of the Chord ring, including node join and leave, file backup and inter-node movement functions.

### Node command

* Lookup(fileName):

  Given a file name, return the address of the file storage node.

* Storefile(fileName): 

  Given a filename, upload a local file to the Chord ring. The file will be scattered with a Chord address based on the filename, and will be encrypted and hosted on the corresponding node according to the storage rules. Since the file is encrypted by the key of the uploading node, the host will not be able to view the file contents.

* Get(fileName): 

  Given a file name, find the location in the Chord ring where the file exists, if the file exists, then download it to the local folder of the current node and decrypt the contents according to the node's key.

* PrintState():

  Print the current node status, including finger table and successor list.

* Quit:

  Shutdown current node.

### File Security and Storage Redundancy

All files are encrypted with the public key of the current node before being uploaded to the chord, so the custodian will not be able to access the file contents. When we download the file, it will be decrypted using the node's private key.

The use of asymmetric encryption algorithms allows this process to be extended to the file sharing process by using a remote RPC method to obtain the target's public key and then encrypt the file, which is decrypted by the shared object using the private key.

Our system also supports storage redundancy, where each file stored in a node's bucket has a backup in its first successor. When the current node crushed out of the chord due to an accident, our system can still ensure that the hosted files can still be accessed by their successors, so that no fatal error of file loss can occur.

### Cautions
* File name should be **unique**. Otherwise, the file store will fail (lazy handling).