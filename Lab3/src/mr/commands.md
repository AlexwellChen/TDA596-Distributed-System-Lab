```golang
go build -race -buildmode=plugin ../mrapps/wc.go
go run -race mrcoordinator.go pg-*.txt
go run -race mrworker.go wc.so
```

Generate correct word count result:
```golang
go run -race mrsequential.go wc.so pg*txt
sort mr-out-0 > mr-correct-wc.txt
```
Generate word count result for mapReduce function:
```golang
rm mr-out*
go run -race mrcoordinator.go pg-*.txt
go run -race mrworker.go wc.so
(can run many workers)
sort mr-out* > mr-wc.txt
```

Compare difference:
`diff mr-wc.txt mr-correct-wc.txt`

在只有一个worker的情况下diff没有问题，test程序启动了四个worker，然后就test fail了