# nad-inject-fn

Nad-inject-fn inhects a network attachement definition from IP allocations

## dev test

arguments

```bash
kpt fn source data | go run main.go
```

## run

```bash
kpt fn eval --type mutator ./data  -i docker.io/yndd/nad-inject-fn:latest 
```
