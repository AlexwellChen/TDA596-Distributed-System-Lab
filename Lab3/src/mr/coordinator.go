package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"path/filepath"
	"sync"
)

const TempDir = "tmp"
const TaskTimeout = 10

type TaskType int
type TaskStatus int
type JobStatus int

const (
	MapTask TaskType = iota
	ReduceTask
	NoTask
	ExitTask
)

const (
	NotStarted TaskStatus = iota
	InProgress
	Completed
)

type Task struct {
	Type     TaskType
	Status   TaskStatus
	Index    int
	Files    []string
	WorkerId int
}

type Host struct {
	Addr string
	Port string
}

type Coordinator struct {
	// Your definitions here.
	hostAddr    string     // host address
	hostPort    string     // host port
	nMap        int        // number of map tasks
	nReduce     int        // number of reduce tasks
	mapTasks    []Task     // map tasks
	reduceTasks []Task     // reduce tasks
	mu          sync.Mutex // lock for accessing shared data
	hosts       []Host     // hosts that are available to run tasks (For remote execution)
}

// Your code here -- RPC handlers for the worker to call.

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", c.hostAddr+":"+c.hostPort)
	// sockname := coordinatorSock()
	// os.Remove(sockname)
	// l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}

	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.
	nMap := len(files)
	c.nMap = nMap
	c.nReduce = nReduce
	c.mapTasks = make([]Task, nMap)
	c.reduceTasks = make([]Task, nReduce)
	c.hostAddr = "localhost"
	c.hostPort = "8080"

	// Initialize map tasks
	for i := 0; i < nMap; i++ {
		mapTask := Task{MapTask, NotStarted, i, []string{files[i]}, -1}
		c.mapTasks = append(c.mapTasks, mapTask)
	}

	// Initialize reduce tasks
	for i := 0; i < nReduce; i++ {
		reduceTask := Task{ReduceTask, NotStarted, i, []string{}, -1}
		c.reduceTasks = append(c.reduceTasks, reduceTask)
	}

	c.server()

	// Create temporary files for reduce tasks
	outFiles, _ := filepath.Glob("mr-out*")
	for _, f := range outFiles {
		if err := os.Remove(f); err != nil {
			fmt.Printf("Cannot remove file %v\n", f)
		}
	}
	err := os.RemoveAll(TempDir)
	if err != nil {
		fmt.Printf("Cannot remove temp directory %v\n", TempDir)
	}
	err = os.Mkdir(TempDir, 0755)
	if err != nil {
		fmt.Printf("Cannot create temp directory %v\n", TempDir)
	}

	return &c
}
