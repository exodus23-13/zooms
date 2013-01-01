package processtree

import (
	"math/rand"
	"os"
	"strconv"
	"syscall"

	"github.com/exodus23-13/zooms/go/messages"
	slog "github.com/exodus23-13/zooms/go/shinylog"
	"github.com/exodus23-13/zooms/go/unixsocket"
)

type SlaveMonitor struct {
	tree             *ProcessTree
	remoteMasterFile *os.File
}

func Error(err string) {
	// TODO
	println(err)
}

func StartSlaveMonitor(tree *ProcessTree, done chan bool) chan bool {
	quit := make(chan bool)
	go func() {
		localMasterFile, remoteMasterFile, err := unixsocket.Socketpair(syscall.SOCK_DGRAM)
		if err != nil {
			Error("Couldn't create socketpair")
		}

		monitor := &SlaveMonitor{tree, remoteMasterFile}

		localMasterSocket, err := unixsocket.NewUsockFromFile(localMasterFile)
		if err != nil {
			Error("Couldn't Open UNIXSocket")
		}

		// We just want this unix socket to be a channel so we can select on it...
		registeringFds := make(chan int, 3)
		go func() {
			for {
				fd, err := localMasterSocket.ReadFD()
				if err != nil {
					slog.Error(err)
				}
				registeringFds <- fd
			}
		}()

		for _, slave := range monitor.tree.SlavesByName {
			go slave.Run(monitor)
		}

		for {
			select {
			case <-quit:
				monitor.cleanupChildren()
				done <- true
				return
			case fd := <-registeringFds:
				go monitor.slaveDidBeginRegistration(fd)
			}
		}
	}()
	return quit
}

func (mon *SlaveMonitor) cleanupChildren() {
	for _, slave := range mon.tree.SlavesByName {
		slave.ForceKill()
	}
}

func (mon *SlaveMonitor) slaveDidBeginRegistration(fd int) {
	// Having just started the process, we expect an IO, which we convert to a UNIX domain socket
	fileName := strconv.Itoa(rand.Int())
	slaveFile := unixsocket.FdToFile(fd, fileName)
	slaveUsock, err := unixsocket.NewUsockFromFile(slaveFile)
	if err != nil {
		slog.Error(err)
	}
	if err = slaveUsock.Conn.SetReadBuffer(1024); err != nil {
		slog.Error(err)
	}
	if err = slaveUsock.Conn.SetWriteBuffer(1024); err != nil {
		slog.Error(err)
	}

	// We now expect the slave to use this fd they send us to send a Pid&Identifier Message
	msg, err := slaveUsock.ReadMessage()
	if err != nil {
		slog.Error(err)
	}
	pid, identifier, err := messages.ParsePidMessage(msg)

	// And the last step before executing its action, the slave sends us a pipe it will later use to
	// send us all the features it's loaded.
	featurePipeFd, err := slaveUsock.ReadFD()
	if err != nil {
		slog.Error(err)
	}

	slaveNode := mon.tree.FindSlaveByName(identifier)
	if slaveNode == nil {
		Error("slavemonitor.go:slaveDidBeginRegistration:Unknown identifier:" + identifier)
	}

	slaveNode.SlaveWasInitialized(pid, slaveUsock, featurePipeFd)
}
