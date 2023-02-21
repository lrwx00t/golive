package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
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
	runner_killer()
	cmd := exec.Command("go", append([]string{"run"}, runnerOpts.args...)...)
	cmd.Dir = runnerOpts.path
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			os.Stdout.WriteString(color.GreenString("STDOUT: %s", scanner.Text()+"\n"))
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			os.Stderr.WriteString(color.RedString("STDERR: %s", scanner.Text()+"\n"))
		}
	}()
}

func delete_empty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func runner_killer() {
	mainPID := os.Getpid()
	// TODO: Hack
	cmd := exec.Command("sh", "-c", fmt.Sprintf("ps aux | grep go-build | grep -v grep | grep -v %s | awk '{ print $2 }'", strconv.Itoa(mainPID)))

	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error running command:", err)
		return
	}
	outputStr := strings.TrimSpace(string(output))

	pids := delete_empty(strings.Split(outputStr, "\n"))
	if len(pids) == 0 {
		return
	}
	for _, pidStr := range pids {
		if pidStr != "" {
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				fmt.Printf("Error parsing PID: %s\n", err)
				return
			}
			killCmd := exec.Command("kill", strconv.Itoa(pid))
			err = killCmd.Start()
			if err != nil {
				fmt.Printf("Error killing process %d: %s\n", pid, err)
			} else {
				err := killCmd.Wait()
				if err != nil {
					fmt.Printf("Error waiting for process %d to exit: %s\n", pid, err)
				} else {
					fmt.Println(color.YellowString("Killed process %d", pid))
				}
			}
		}
	}
	runner_killer()
}

func main() {
	path := flag.String("path", ".", "go project path")
	pkg := flag.String("package", ".", "go package or file name")
	flag.Parse()

	log.Println("golive started 👀")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("failed to create watcher: %s", err)
	}
	defer watcher.Close()

	t := time.Now()

	go func() {
		wg := sync.WaitGroup{}
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op.Has(fsnotify.Chmod) || strings.HasSuffix(event.Name, ".swp") {
					continue
				}
				if time.Since(t).Seconds() <= 1.0 {
					continue
				}
				t = time.Now()
				wg.Add(1)
				go runner(context.Background(), &wg, RunnerOpts{
					path: *path,
					args: []string{*pkg},
				})
				wg.Wait()
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

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	<-ctx.Done()
	log.Println("shutting down golive gracefully. Bye 👋")
	if err := watcher.Close(); err != nil {
		log.Printf("failed to close watcher: %s", err)
	}
}
