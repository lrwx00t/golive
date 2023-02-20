package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
)

type RunnerOpts struct {
	path string
	args []string
}

func runner(ctx context.Context, wg *sync.WaitGroup, runnerOpts RunnerOpts) {
	defer wg.Done()
	cmd := exec.Command("go", append([]string{"run"}, runnerOpts.args...)...)
	cmd.Dir = runnerOpts.path
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error: %s", err)
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
		log.Fatalf("path cannot be empty")
	}

	log.Println("golive started 👀")

	// create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("failed to create watcher: %s", err)
	}
	defer watcher.Close()

	// Start listening for events.
	go func() {
		wg := sync.WaitGroup{}
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op.Has(fsnotify.Chmod) {
					continue
				}
				wg.Wait()
				wg.Add(1)
				go runner(context.Background(), &wg, RunnerOpts{
					path: *path,
					args: []string{*pkg},
				})
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("watcher error: %s", err)
			}
		}
	}()

	err = watcher.Add(*path)
	if err != nil {
		log.Fatalf("failed to add path to watcher: %s", err)
	}

	// Handle SIGINT signal to gracefully shut down the application.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	log.Println("shutting down golive gracefully. Bye 👋")
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := watcher.Close(); err != nil {
		log.Printf("failed to close watcher: %s", err)
	}
}
