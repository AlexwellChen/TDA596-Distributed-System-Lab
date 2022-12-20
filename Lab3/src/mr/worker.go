package mr

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"time"
)
const TaskInterval = 200
var nReduce int

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.
	n, ok := getNReduce()
	if !ok {
		fmt.Println("Cannot get nReduce from coordinator")
	}
	nReduce = n
	// uncomment to send the Example RPC to the coordinator.
	// CallExample()
	for {
		reply,ok := requestTask()
		if !ok {
			fmt.Println("Cannot request task from coordinator")
			return
		}
		if reply.TaskType == ExitTask {
			fmt.Println("No more tasks to do, worker exit")
			return
		}
		exit, ok := false , true
		if reply.TaskType == NoTask {
			fmt.Println("All map or reduce tasks are in progress, worker wait")
		}else if reply.TaskType == MapTask {
			doMap(mapf,reply.TaskFile,reply.TaskId)
			exit, ok = completeTask(MapTask,reply.TaskId)
		}else if reply.TaskType == ReduceTask {
			doReduce(reducef,reply.TaskId)
			exit, ok = completeTask(ReduceTask,reply.TaskId)
		}

		if !ok || exit {
			fmt.Println("Coordinator exit or all tasks complete, worker exit")
			return
		}

		time.Sleep(TaskInterval * time.Millisecond)
	}

}

func doMap(mapf func(string, string) []KeyValue,filepath string,mapId int) {
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("cannot open %v", filepath)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("cannot read %v", filepath)
	}
	kv := mapf(filepath, string(content))
	writeMapOutput(kv,mapId)
}

func writeMapOutput(kv []KeyValue,mapId int) {
}

func doReduce(reducef func(string, []string) string,reduceId int) {
}

func writeReduceOutput(reducef func(string,[]string) string, kvMap map[string][]string,reduceId int) {
}

func getNReduce() (int,bool) {
	args := GetNReduceArgs{}
	reply := GetNReduceReply{}

	ok := call("Coordinator.GetNReduce", &args, &reply)
	return reply.NReduce,ok
}

func requestTask() (*RequestTaskReply,bool) {
	args := RequestTaskArgs{}
	args.WorkerId = os.Getpid()
	reply := RequestTaskReply{}

	ok := call("Coordinator.RequestTask", &args, &reply)
	return &reply,ok
}

func completeTask(taskType TaskType,taskId int) (bool,bool) {
	args := CompleteTaskArgs{}
	args.TaskType = taskType
	args.TaskId = taskId
	args.WorkerId = os.Getpid()
	reply := CompleteTaskReply{}

	ok := call("Coordinator.CompleteTask", &args, &reply)
	return reply.CanExit,ok
}
// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
