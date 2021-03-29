Proof-of-concept shell interface outside a terminal.

This works with bash and the Go sh/interp package.

![](screenshot.png) 

Build with:
```shell
go generate
go build
```

Shift-Enter to enter a new line.\
Ctrl-C to interrupt a running command.\
Double-click to collapse a command's output.

It's horrible in many ways. Most notably, the shell interpreter instance
is global.