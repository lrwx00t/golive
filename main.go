package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
)

type RunnerOpts struct {
	done *chan bool
	path string
	args []string
}

func runner(runnerOpts RunnerOpts) {
	defer func() {
		*runnerOpts.done <- true
	}()
	cmd := exec.Command("go", append([]string{"run"}, runnerOpts.args...)...)
	cmd.Dir = runnerOpts.path
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(color.RedString("Error: %v", err))
		fmt.Print(color.RedString("%s", out))
		return
	}
	fmt.Print(color.GreenString("%s", out))
}

func main() {
	path := flag.String("path", "", "go project path")
	pkg := flag.String("package", ".", "go package or file name")
	flag.Parse()
	if len(*path) == 0 {
		log.Fatal("path cannot be empty")
	}
	done := make(chan bool)
	runnerOpts := RunnerOpts{
		done: &done,
		path: *path,
		args: []string{*pkg},
	}
	log.Println("golive started 👀..")

	// create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Start listening for events.
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				fmt.Println(event)
				if !ok {
					return
				}
				if event.Op.Has(fsnotify.Chmod) {
					continue
				}
				go runner(runnerOpts)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
			<-*runnerOpts.done
		}
	}()

	err = watcher.Add(runnerOpts.path)
	if err != nil {
		log.Fatal(err)
	}

	// Block main goroutine forever.
	<-make(chan struct{})
}
