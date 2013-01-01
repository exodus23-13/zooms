package filemonitor

import (
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	slog "github.com/exodus23-13/zooms/go/shinylog"
)

// {{{ public API
func Start(done chan bool) (filesChanged chan string, quit chan bool) {
	quit = make(chan bool)
	filesChanged = make(chan string, 1024)
	go start(filesChanged, done, quit)
	return
}

func AddFile(file string) {
	filesToWatch <- file
}

// }}}

var filesToWatch chan string

var watcherIn io.WriteCloser
var watcherOut io.ReadCloser
var watcherErr io.ReadCloser

var fileMutex sync.Mutex

var allWatchedFiles map[string]bool

func start(filesChanged chan string, done, quit chan bool) {
	// this is large because as long as we start
	// watching the files eventually, it's more of a priority to
	// get the slaves booted as quickly as possible.
	filesToWatch = make(chan string, 8192)
	allWatchedFiles = make(map[string]bool)

	cmd := startWrapper(filesChanged)

	for {
		select {
		case <-quit:
			cmd.Process.Kill()
			done <- true
			return
		case path := <-filesToWatch:
			go handleLoadedFileNotification(path)
		}
	}
}

func executablePath() string {
	switch runtime.GOOS {
	case "darwin":
		return path.Join(path.Dir(os.Args[0]), "fsevents-wrapper")
	case "linux":
		gemRoot := path.Dir(path.Dir(os.Args[0]))
		return path.Join(gemRoot, "ext/inotify-wrapper/inotify-wrapper")
	}
	terminate("Unsupported OS")
	return ""
}

func startWrapper(filesChanged chan string) *exec.Cmd {
	cmd := exec.Command(executablePath())
	var err error
	if watcherIn, err = cmd.StdinPipe(); err != nil {
		terminate(err.Error())
	}
	if watcherOut, err = cmd.StdoutPipe(); err != nil {
		terminate(err.Error())
	}
	if watcherErr, err = cmd.StderrPipe(); err != nil {
		terminate(err.Error())
	}

	cmd.Start()

	go func() {
		buf := make([]byte, 2048)
		for {
			n, err := watcherOut.Read(buf)
			if err != nil {
				time.Sleep(500 * time.Millisecond)
				slog.Red("Failed to read from FileSystem watcher process: " + err.Error())
			}
			message := strings.TrimSpace(string(buf[:n]))
			files := strings.Split(message, "\n")
			for _, file := range files {
				filesChanged <- file
			}
		}
	}()

	go func() {
		err := cmd.Wait()
		// gross, but this is an easy way to work around the case where
		// signal propagation hits the wrapper before the master disables logging
		time.Sleep(100 * time.Millisecond)
		terminate("The FS watcher process crashed: " + err.Error())
	}()

	return cmd
}

func handleLoadedFileNotification(file string) {
	fileMutex.Lock()
	// a slave loaded a file. It's up to us here to make sure this file is watched.
	if !allWatchedFiles[file] {
		allWatchedFiles[file] = true
		startWatchingFile(file)
	}
	fileMutex.Unlock()
}

func startWatchingFile(file string) {
	_, err := watcherIn.Write([]byte(file + "\n"))
	if err != nil {
		slog.Error(err)
	}
}

func terminate(message string) {
	slog.Red(message)
	println(message)
	proc, _ := os.FindProcess(os.Getpid())
	proc.Signal(syscall.SIGTERM)
}
