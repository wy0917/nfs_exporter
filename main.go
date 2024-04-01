package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var verbose bool

func debug(format string, v ...interface{}) {
	if verbose {
		fmt.Printf(format+"\n", v...)
	}
}

func writeToFile(ctx context.Context, mountPoint string, ch chan string) {
	start := time.Now()
	file, err := os.Create(mountPoint + "/_testfile")
	if err != nil {
		debug("Failed to create test file at %s: %v", mountPoint, err)
		ch <- fmt.Sprintf(`nfs_write_success{mount_point="%s"} 0`, mountPoint)
		return
	}
	defer file.Close()

	select {
	case <-ctx.Done():
		duration := time.Since(start).Seconds()
		ch <- fmt.Sprintf(`nfs_write_time_seconds{mount_point="%s"} %f`, mountPoint, duration)
		ch <- fmt.Sprintf(`nfs_write_success{mount_point="%s"} 0`, mountPoint)
	default:
		duration := time.Since(start).Seconds()
		ch <- fmt.Sprintf(`nfs_write_time_seconds{mount_point="%s"} %f`, mountPoint, duration)
		ch <- fmt.Sprintf(`nfs_write_success{mount_point="%s"} 1`, mountPoint)
	}
}

func main() {
	var outputPath string
	flag.StringVar(&outputPath, "o", "", "Output file path")
	flag.BoolVar(&verbose, "V", false, "Print debug information")
	flag.Parse()

	out, _ := exec.Command("df", "-PT").Output()
	lines := strings.Split(string(out), "\n")
	ch := make(chan string, len(lines))
	var wg sync.WaitGroup
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 1 && fields[0] != "Filesystem" {
			debug("found%d mountpoint %s fstype %s", len(fields), fields[0], fields[1])
		}
		if len(fields) > 1 && fields[1] == "nfs" {
			wg.Add(1)
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			go func() {
				writeToFile(ctx, fields[6], ch)
				wg.Done()
			}()
		}
	}
	wg.Wait()
	close(ch)
	metrics := make([]string, 0, len(ch))
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	var writer io.Writer = os.Stdout
	if outputPath != "" {
		file, err := os.Create(outputPath)
		if err != nil {
			debug("Error creating file: %v", err)
			os.Exit(1)
		}
		defer file.Close()
		writer = file
	}

	_, err := writer.Write([]byte(strings.Join(metrics, "\n") + "\n"))
	if err != nil {
		debug("Error writing output: %v", err)
		os.Exit(1)
	}
}
