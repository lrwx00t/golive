# golive

`golive` is a tool for real-time Go development, allowing you to see changes to your code as you make them. It automatically runs your code as you work and keeps everything in sync. It currently provides limited support to `go run` only for short and long running processes.

Whenever a change is detected in the directory, such as an updated or newly added file, the code will be executed automatically. If the execution is successful, the output will be displayed in green, otherwise any errors that occurred will be displayed in red. In all cases, `golive` is expected to continue running indefinitely until manually terminated by issuing a `SIGINT` signal e.g. `Ctrl + C`.

<img width="943" alt="image" src="https://user-images.githubusercontent.com/96939525/220201310-2a88c9f3-3377-4efd-b483-abf8cd890b4d.png">


## Install

```bash
go install github.com/lrwx00t/golive
```

## Example

```bash
golive --path ~/src/go-dev/playground/demo
2023/02/20 16:25:36 golive started 👀..

# without any arguments, it defaults to current path
golive
```
