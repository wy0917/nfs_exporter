package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	fstab "github.com/d-tux/go-fstab"
)

var verbose bool
var supportedFsTypes = [...]string{"nfs", "cifs"}

func isSupportedFsType(fstype string) bool {
	for _, supportedFsType := range supportedFsTypes {
		if fstype == supportedFsType {
			return true
		}
	}
	return false
}

func debug(format string, v ...interface{}) {
	if verbose {
		fmt.Printf(format+"\n", v...)
	}
}

func writeToFile(ctx context.Context, mountPoint string, ch chan string, filename string) {
	start := time.Now()
	filePath := filepath.Join(mountPoint, filename)
	file, err := os.Create(filePath)
	if err != nil {
		debug("Failed to create test file at %s: %v", mountPoint, err)
		ch <- fmt.Sprintf(`nfs_write_success{mount_point="%s"} 0`, mountPoint)
		return
	}
	defer func() {
		file.Close()
		if err := os.Remove(filePath); err != nil {
			debug("Failed to delete test file at %s: %v", mountPoint, err)
		}
	}()

	if _, err = file.WriteString(time.Now().String()); err != nil {
		debug("Failed to write to test file at %s: %v", mountPoint, err)
		ch <- fmt.Sprintf(`nfs_write_success{mount_point="%s"} 0`, mountPoint)
		return
	}

	success := 1
	select {
	case <-ctx.Done():
		success = 0
	default:
	}

	duration := time.Since(start).Seconds()
	ch <- fmt.Sprintf(`nfs_write_time_seconds{mount_point="%s"} %f`, mountPoint, duration)
	ch <- fmt.Sprintf(`nfs_write_success{mount_point="%s"} %d`, mountPoint, success)
}

func getMountPoints(fstabPath string) []string {
	mounts, err := fstab.ParseFile(fstabPath)
	if err != nil {
		debug("Failed to parse %s: %v", fstabPath, err)
		panic(err)
	}

	mount_list := []string{}
	for _, mount := range mounts {
		if isSupportedFsType(mount.VfsType) {
			mount_list = append(mount_list, mount.File)
		}
	}
	return mount_list
}

func getMountedPoints() []string {
	out, _ := exec.Command("df", "-PT").Output()
	lines := strings.Split(string(out), "\n")
	mount_list := []string{}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 1 && isSupportedFsType(fields[1]) {
			mount_list = append(mount_list, fields[6])
		}
	}
	return mount_list
}

func main() {
	var outputPath string
	var filename string
	var timeout int
	flag.StringVar(&outputPath, "o", "", "Output file path")
	flag.StringVar(&filename, "f", ".testfile", "The name of the test file")
	flag.IntVar(&timeout, "t", 200, "Timeout in milliseconds")
	flag.BoolVar(&verbose, "V", false, "Print debug information")
	flag.Parse()

	mountPoints := getMountPoints("/etc/fstab")
	mountedPoints := getMountedPoints()

	ch := make(chan string, len(mountPoints)+len(mountedPoints))

	for _, mp := range mountPoints {
		if !slices.Contains(mountedPoints, mp) {
			// mount point does not mounted
			ch <- fmt.Sprintf(`nfs_write_success{mount_point="%s"} 0`, mp)
		}
	}

	var wg sync.WaitGroup
	for _, mountPoint := range mountedPoints {
		wg.Add(1)
		go func(mountPoint string) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
			defer cancel()
			writeToFile(ctx, mountPoint, ch, filename)
			wg.Done()
		}(mountPoint)
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
