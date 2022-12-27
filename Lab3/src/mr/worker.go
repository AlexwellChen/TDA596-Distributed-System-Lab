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
	"path/filepath"
	"sort"
	"net/http"
	"time"
)

const TaskInterval = 200

var nReduce int
var coordinator_addr string
var server_addr string
var if_cloud bool

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
	reducef func(string, []string) string, position string) {

	// Your worker implementation here.
	if position == "cloud" {
		// wanqiu's aws address here
		coordinator_addr = "3.230.196.47:8000"
		// qi's server address
		server_addr = "http://3.213.15.92:8080/root/"
		if_cloud = true
	} else if position == "local"{
		coordinator_addr = "localhost:8000"
		if_cloud = false
	}else{
		fmt.Println("Wrong argument")
		return
	}
	n, ok := getNReduce()
	if !ok {
		fmt.Println("Cannot get nReduce from coordinator")
	}
	nReduce = n
	// uncomment to send the Example RPC to the coordinator.
	// CallExample()
	fmt.Println("Worker start")
	for {
		reply, ok := requestTask()
		if !ok {
			fmt.Println("Cannot request task from coordinator")
			return
		}
		if reply.TaskType == ExitTask {
			// fmt.Println("No more tasks to do, worker exit")
			return
		}
		exit, ok := false, true
		if reply.TaskType == NoTask {
			// fmt.Println("All map or reduce tasks are in progress, worker wait")
			// do nothing
		} else if reply.TaskType == MapTask {
			if reply.TaskFile == "" {
				fmt.Println("No map task file to do")
			} else {
				doMap(mapf, reply.TaskFile, reply.TaskId)
				exit, ok = completeTask(MapTask, reply.TaskId)
			}
		} else if reply.TaskType == ReduceTask {
			doReduce(reducef, reply.TaskId)
			exit, ok = completeTask(ReduceTask, reply.TaskId)
		}

		if !ok || exit {
			// fmt.Println("Coordinator exit or all tasks complete, worker exit")
			return
		}

		time.Sleep(TaskInterval * time.Millisecond)
	}

}

func doMap(mapf func(string, string) []KeyValue, filepath string, mapId int) {
	var content []byte
	if if_cloud {
		// if running on different machines, get file from cloud server
		res, err := http.Get(server_addr + filepath)
		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()
		content, err = ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		// if running on local machine, read file from local
		file, err := os.Open(filepath)
		if err != nil {
			fmt.Printf("cannot open %v\n", filepath)
		}
		content, err = ioutil.ReadAll(file)
		if err != nil {
			fmt.Printf("cannot read %v\n", filepath)
		}
	}
	kv := mapf(filepath, string(content))
	writeMapOutput(kv, mapId)
}

func writeMapOutput(kv []KeyValue, mapId int) {
	prefix := fmt.Sprintf("%v/mr-%v-", TempDir, mapId)
	files := make([]*os.File, 0, nReduce)
	writers := make([]*bufio.Writer, 0, nReduce)
	encoders := make([]*json.Encoder, 0, nReduce)

	for i := 0; i < nReduce; i++ {
		filePath := fmt.Sprintf("%v-%v-%v", prefix, i, os.Getpid())
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Printf("cannot create %v\n", filePath)
		}
		writer := bufio.NewWriter(file)
		files = append(files, file)
		writers = append(writers, writer)
		encoders = append(encoders, json.NewEncoder(writer))
	}

	//write map output kv to files
	for _, kv := range kv {
		id := ihash(kv.Key) % nReduce
		err := encoders[id].Encode(&kv)
		if err != nil {
			fmt.Printf("cannot encode %v to file\n", kv)
		}
	}

	//flush all files
	for i, writer := range writers {
		err := writer.Flush()
		if err != nil {
			fmt.Printf("cannot flush for file: %v\n", files[i].Name())
		}
	}

	//rename files
	for i, file := range files {
		file.Close()
		newPath := fmt.Sprintf("%v-%v", prefix, i)
		err := os.Rename(file.Name(), newPath)
		if err != nil {
			fmt.Printf("cannot rename %v to %v\n", file.Name(), newPath)
		}
		// If running on different machines, upload file to cloud server
		if if_cloud {
			// read file content
			pfile, err := os.Open(newPath)
			if err != nil {
				fmt.Printf("cannot open %v\n", newPath)
			}
			content, err := ioutil.ReadAll(pfile)
			if err != nil {
				fmt.Printf("cannot read %v\n", newPath)
			}
			reader := bytes.NewReader(content)
			// TODO: check what we did in lab1 for post request
			res,err:=http.Post(server_addr+newPath,"text/plain;charset=UTF-8",reader)
		}
	}

}

func doReduce(reducef func(string, []string) string, reduceId int) {
	if if_cloud {
		// if running on different machines, get files from cloud server
		// get root/tmp file list
		// match file name pattern "tmp/mr-*-reduceId"
		all_files, err := http.Get(server_addr + TempDir)
		if err != nil {
			log.Fatal(err)
		}
		defer all_files.Body.Close()
		content, err := ioutil.ReadAll(all_files.Body)
		if err != nil {
			log.Fatal(err)
		}
		files := make([]string, 0)
		for _, file := range strings.Split(string(content), " ") {
			if strings.HasPrefix(file, "mr-") && strings.HasSuffix(file, fmt.Sprintf("-%v", reduceId)) {
				files = append(files, file)
			}
		}
	} else {
		files, err := filepath.Glob(fmt.Sprintf("%v/mr-*-%v", TempDir, reduceId))
		if err != nil {
			fmt.Printf("cannot find files for reduceId: %v\n", reduceId)
		}
	}
	kvMap := make(map[string][]string)
	var kv KeyValue
	for _, filePath := range files {
		if if_cloud {
			// if running on different machines, get file from cloud server
			file, err := http.Get(server_addr + filePath)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Body.Close()
			content, err := ioutil.ReadAll(file.Body)
			if err != nil {	
				log.Fatal(err)
			}
			decoder := json.NewDecoder(bytes.NewReader(content))
		}else{
			file, err := os.Open(filePath)
			if err != nil {
				fmt.Printf("cannot open %v\n", filePath)
			}
			decoder := json.NewDecoder(file)
		}
		for decoder.More() {
			err := decoder.Decode(&kv)
			if err != nil {
				fmt.Printf("cannot decode %v\n", filePath)
			}
			kvMap[kv.Key] = append(kvMap[kv.Key], kv.Value)
		}
	}
	writeReduceOutput(reducef, kvMap, reduceId)
}

func writeReduceOutput(reducef func(string, []string) string, kvMap map[string][]string, reduceId int) {

	//sort keyvalue map
	keys := make([]string, 0, len(kvMap))
	for key := range kvMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	//create file
	filePath := fmt.Sprintf("%v/mr-out-%v-%v", TempDir, reduceId, os.Getpid())
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("cannot create %v\n", filePath)
	}

	//write to file
	for _, key := range keys {
		value := reducef(key, kvMap[key])
		// Delete space here
		_, err := fmt.Fprintf(file, "%v %v\n", key, value)
		if err != nil {
			fmt.Printf("cannot write (%v,%v) to file: %v\n", key, value, filePath)
		}
	}

	//rename file
	file.Close()
	newPath := fmt.Sprintf("mr-out-%v", reduceId)
	err = os.Rename(filePath, newPath)
	if err != nil {
		fmt.Printf("cannot rename file : %v to %v\n", filePath, newPath)
	}
	if(if_cloud){
		// upload file to cloud server
		// read file content
		pfile, err := os.Open(newPath)
		if err != nil {
			fmt.Printf("cannot open %v\n", newPath)
		}
		content, err := ioutil.ReadAll(pfile)
		if err != nil {
			fmt.Printf("cannot read %v\n", newPath)
		}
		reader := bytes.NewReader(content)
		res,err:=http.Post(server_addr+newPath,"text/plain;charset=UTF-8",reader)
		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			log.Fatal("cannot upload file to cloud server")
		}
	}
}

func getNReduce() (int, bool) {
	args := GetNReduceArgs{}
	reply := GetNReduceReply{}

	ok := call("Coordinator.GetNReduce", &args, &reply)
	return reply.NReduce, ok
}

func requestTask() (*RequestTaskReply, bool) {
	args := RequestTaskArgs{}
	args.WorkerId = os.Getpid()
	reply := RequestTaskReply{}

	ok := call("Coordinator.RequestTask", &args, &reply)
	return &reply, ok
}

func completeTask(taskType TaskType, taskId int) (bool, bool) {
	args := CompleteTaskArgs{}
	args.TaskType = taskType
	args.TaskId = taskId
	args.WorkerId = os.Getpid()
	reply := CompleteTaskReply{}

	ok := call("Coordinator.CompleteTask", &args, &reply)
	return reply.CanExit, ok
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
	// sockname := coordinatorSock()
	c, err := rpc.DialHTTP("tcp", coordinator_addr)
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
