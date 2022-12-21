```golang
go build -race -buildmode=plugin ../mrapps/wc.go
go run -race mrcoordinator.go pg-*.txt
go run -race mrworker.go wc.so
```

Compare difference:
`diff mr-tmp/mr-wc-all mr-tmp/mr-correct-wc.txt`
