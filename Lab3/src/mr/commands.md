```golang
go build -race -buildmode=plugin ../mrapps/wc.go
go run -race mrcoordinator.go pg-*.txt cloud
go run -race mrworker.go wc.so cloud
```

Compare difference:
`diff mr-tmp/mr-wc-all mr-tmp/mr-correct-wc.txt`
