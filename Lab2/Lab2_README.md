### Lab2 Chord 

**RPC方法编程规范**

所有RPC方法request不限制类型，返回类型必须为reply XXXReply，其中XXX为RPC方法名，如GetSuccessorReply。如果不使用结构体包装会出现无法被jsonrpc解析的情况。