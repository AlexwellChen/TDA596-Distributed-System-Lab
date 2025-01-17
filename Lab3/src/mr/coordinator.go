package mr

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const TempDir = "tmp"
const TaskTimeout = 10

type TaskType int
type TaskStatus int
type JobStatus int

var run_position string

const (
	// iota is enum
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
	File     string // File location in shared file system for tasks, also need to compatible with Global File System
	WorkerId int
}

type Host struct {
	Addr string
	Port string
}

type Coordinator struct {
	// Your definitions here.
	hostAddr         string     // host address
	hostPort         string     // host port
	nMap             int        // number of map tasks
	nReduce          int        // number of reduce tasks
	nMapCompleted    int        // number of map tasks completed
	nReduceCompleted int        // number of reduce tasks completed
	mapTasks         []Task     // map tasks
	reduceTasks      []Task     // reduce tasks
	mu               sync.Mutex // lock for accessing shared data
	// hosts       []Host     // hosts that are available to run tasks (For remote execution)
}

// Your code here -- RPC handlers for the worker to call.

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) GetNReduce(args *GetNReduceArgs, reply *GetNReduceReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	reply.NReduce = len(c.reduceTasks)
	return nil
}

/*-------------------------------------------------------*/
/*-------------------- Task RPC function ----------------*/
/*-------------------------------------------------------*/

func (c *Coordinator) RequestTask(args *RequestTaskArgs, reply *RequestTaskReply) error {

	task := c.selectTask()
	// return reference in order to write workerId to tasks
	c.mu.Lock()
	defer c.mu.Unlock()
	task.WorkerId = args.WorkerId

	reply.TaskType = task.Type
	reply.TaskId = task.Index
	reply.TaskFile = task.File
	if run_position == "cloud" {
		fmt.Println("Dispatching task", task.Type, task.Index, "to worker", args.WorkerId)
	}
	// wait for task to complete only for map and reduce tasks
	if task.Type == MapTask || task.Type == ReduceTask {
		go c.waitForTask(task)
	}

	return nil
}

func (c *Coordinator) CompleteTask(args *CompleteTaskArgs, reply *CompleteTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var task *Task
	if args.TaskType == MapTask {
		task = &c.mapTasks[args.TaskId]
	} else if args.TaskType == ReduceTask {
		task = &c.reduceTasks[args.TaskId]
	} else {
		fmt.Printf("CompleteTask: Invalid task type %v\n", args.TaskType)
		return nil
	}
	if task.Status == InProgress && task.WorkerId == args.WorkerId {
		task.Status = Completed
		// 这里不能直接减，selectTask还要用到nMap和nReduce的值
		// 如果nmap--会出现只分配前面几个任务的情况
		// Solution: 用另外两个变量记录已经完成的任务数
		if args.TaskType == MapTask {
			// c.nMap--
			c.nMapCompleted++
		} else if args.TaskType == ReduceTask {
			// c.nReduce--
			c.nReduceCompleted++
		}
	}

	reply.CanExit = c.nMapCompleted == c.nMap && c.nReduceCompleted == c.nReduce

	return nil
}

func (c *Coordinator) selectTask() *Task {

	c.mu.Lock()
	defer c.mu.Unlock()

	// Dispatch map tasks first
	for i := 0; i < c.nMap; i++ {
		if c.mapTasks[i].Status == NotStarted {
			c.mapTasks[i].Status = InProgress
			c.mapTasks[i].Index = i
			return &c.mapTasks[i]
		}
	}
	if c.nMapCompleted != c.nMap {
		return &Task{NoTask, NotStarted, -1, "", -1}
	} else {
		// Dispatch reduce tasks only if all map tasks are completed
		for i := 0; i < c.nReduce; i++ {
			if c.reduceTasks[i].Status == NotStarted {
				c.reduceTasks[i].Status = InProgress
				c.reduceTasks[i].Index = i
				return &c.reduceTasks[i]
			}
		}
	}
	if c.nReduceCompleted != c.nReduce {
		return &Task{NoTask, NotStarted, -1, "", -1}
	} else {
		return &Task{ExitTask, NotStarted, -1, "", -1}
	}
}

func (c *Coordinator) waitForTask(task *Task) {
	if task.Type != MapTask && task.Type != ReduceTask {
		fmt.Println("waitForTask: Invalid task type ", task.Type)
		return
	}

	// Wait for task to complete
	<-time.After(TaskTimeout * time.Second)

	c.mu.Lock()
	defer c.mu.Unlock()

	// If task is still in progress, mark it as not started
	if task.Status == InProgress {
		task.Status = NotStarted
		task.WorkerId = -1
		fmt.Println("Task timed out, reset task status:", task.Index)
	}
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	var l net.Listener
	var e error
	if run_position == "cloud" {
		l, e = net.Listen("tcp", c.hostAddr+":"+c.hostPort)
	} else {
		sockname := coordinatorSock()
		os.Remove(sockname)
		l, e = net.Listen("unix", sockname)
	}
	if e != nil {
		log.Fatal("listen error:", e)
	}

	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false
	// If all tasks are completed, return true
	c.mu.Lock()
	defer c.mu.Unlock()
	ret = c.nMap == c.nMapCompleted && c.nReduce == c.nReduceCompleted
	if ret && run_position == "cloud" {
		files_list := getCloudFileList("mr-out*")
		for i := 0; i < len(files_list); i++ {
			// Download files from cloud
			res, err := http.Get("http://3.213.15.92:8080/root/" + files_list[i])
			if err != nil {
				log.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusOK {
				log.Fatal("Error in downloading file")
			}
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Fatal(err)
			}
			// Write to local file
			f, err := os.Create(files_list[i])
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			_, err = f.Write(body)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	return ret
}
func getCloudFileList(prefix string) []string {
	res, err := http.Get("http://3.213.15.92:8080/root")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Fatal("Error in getting file list")
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	file_list := strings.Split(string(body), " ")
	new_file_list := make([]string, 0)
	for i := 0; i < len(file_list); i++ {
		// if file_list[i] matches with pg-*.txt then append to files
		// Use regular expression to match
		matched, err := regexp.MatchString(prefix, file_list[i])
		if err != nil {
			log.Fatal(err)
		}
		if matched {
			new_file_list = append(new_file_list, file_list[i])
			fmt.Println(file_list[i])
		}
	}
	return new_file_list
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int, position string) *Coordinator {
	c := Coordinator{}

	// Your code here.
	// Connect to server and get a list of files
	if position == "cloud" {
		run_position = "cloud"
		new_file_list := getCloudFileList("pg-*")
		files = new_file_list
	} else {
		run_position = "local"
	}
	nMap := len(files)
	c.nMap = nMap
	c.nReduce = nReduce
	c.mapTasks = make([]Task, nMap)
	c.reduceTasks = make([]Task, nReduce)
	if position == "cloud" {
		c.hostAddr = "0.0.0.0"
		c.hostPort = "8000"
	} else {
		c.hostAddr = "localhost"
		c.hostPort = "8000"
	}

	// Initialize map tasks
	for i := 0; i < nMap; i++ {
		mapTask := Task{MapTask, NotStarted, i, files[i], -1}
		// Append does not work here, c.mapTasks[i] is empty. Switched to assign value
		c.mapTasks[i] = mapTask
	}

	// Initialize reduce tasks
	for i := 0; i < nReduce; i++ {
		reduceTask := Task{ReduceTask, NotStarted, i, "", -1}
		c.reduceTasks[i] = reduceTask
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
