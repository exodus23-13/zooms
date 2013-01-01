package clienthandler

import (
	"errors"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/exodus23-13/zooms/go/messages"
	"github.com/exodus23-13/zooms/go/processtree"
	slog "github.com/exodus23-13/zooms/go/shinylog"
	"github.com/exodus23-13/zooms/go/unixsocket"
	"github.com/exodus23-13/zooms/go/zerror"
)

const zoomsSockName string = ".zooms.sock"

func Start(tree *processtree.ProcessTree, done chan bool) chan bool {
	quit := make(chan bool)
	go func() {
		path, _ := filepath.Abs(zoomsSockName)
		addr, err := net.ResolveUnixAddr("unix", path)
		if err != nil {
			zerror.Error("Can't open socket.")
		}
		listener, err := net.ListenUnix("unix", addr)
		if err != nil {
			zerror.ErrorCantCreateListener()
		}

		connections := make(chan *unixsocket.Usock)
		go func() {
			for {
				if conn, err := listener.AcceptUnix(); err != nil {
					zerror.ErrorUnableToAcceptSocketConnection()
					time.Sleep(500 * time.Millisecond)
				} else {
					connections <- unixsocket.NewUsock(conn)
				}
			}
		}()

		for {
			select {
			case <-quit:
				listener.Close()
				done <- true
				return
			case conn := <-connections:
				go handleClientConnection(tree, conn)
			}
		}
	}()
	return quit
}

// see docs/client_master_handshake.md
func handleClientConnection(tree *processtree.ProcessTree, usock *unixsocket.Usock) {
	defer usock.Close()
	// we have established first contact to the client.

	command, clientPid, arguments, err := receiveCommandArgumentsAndPid(usock, nil)
	commandNode, slaveNode, err := findCommandAndSlaveNodes(tree, command, err)
	if err != nil {
		// connection was established, no data was sent. Ignore.
		return
	}
	command = commandNode.Name // resolve aliases

	clientFile, err := receiveTTY(usock, err)
	defer clientFile.Close()

	if err == nil && slaveNode.Error != "" {
		// we can skip steps 3-5 as they deal with the command process we're not spawning.
		// Write a fake pid (step 6)
		usock.WriteMessage("0")
		// Write the error message to the terminal
		clientFile.Write([]byte(slaveNode.Error))
		// Skip step 7, and write an exit code to the client (step 8)
		usock.WriteMessage("1")
		return
	}

	commandUsock, err := bootNewCommand(slaveNode, command, err)
	defer commandUsock.Close()

	err = sendClientPidAndArgumentsToCommand(commandUsock, clientPid, arguments, err)

	err = sendTTYToCommand(commandUsock, clientFile, err)

	cmdPid, err := receivePidFromCommand(commandUsock, err)

	err = sendCommandPidToClient(usock, cmdPid, err)

	exitStatus, err := receiveExitStatus(commandUsock, err)

	err = sendExitStatus(usock, exitStatus, err)

	if err != nil {
		slog.Error(err)
	}
	// Done! Hooray!
}

func receiveCommandArgumentsAndPid(usock *unixsocket.Usock, err error) (string, int, string, error) {
	if err != nil {
		return "", -1, "", err
	}

	msg, err := usock.ReadMessage()
	if err != nil {
		return "", -1, "", err
	}
	command, clientPid, arguments, err := messages.ParseClientCommandRequestMessage(msg)
	if err != nil {
		return "", -1, "", err
	}

	return command, clientPid, arguments, err
}

func findCommandAndSlaveNodes(tree *processtree.ProcessTree, command string, err error) (*processtree.CommandNode, *processtree.SlaveNode, error) {
	if err != nil {
		return nil, nil, err
	}

	commandNode := tree.FindCommand(command)
	if commandNode == nil {
		return nil, nil, errors.New("ERROR: Node not found!: " + command)
	}
	command = commandNode.Name
	slaveNode := commandNode.Parent

	return commandNode, slaveNode, nil
}

func receiveTTY(usock *unixsocket.Usock, err error) (*os.File, error) {
	if err != nil {
		return nil, err
	}

	clientFd, err := usock.ReadFD()
	if err != nil {
		return nil, errors.New("Expected FD, none received!")
	}
	fileName := strconv.Itoa(rand.Int())
	clientFile := unixsocket.FdToFile(clientFd, fileName)

	return clientFile, nil
}

func sendClientPidAndArgumentsToCommand(commandUsock *unixsocket.Usock, clientPid int, arguments string, err error) error {
	if err != nil {
		return err
	}

	msg := messages.CreatePidAndArgumentsMessage(clientPid, arguments)
	_, err = commandUsock.WriteMessage(msg)
	return err
}

func receiveExitStatus(commandUsock *unixsocket.Usock, err error) (string, error) {
	if err != nil {
		return "", err
	}

	return commandUsock.ReadMessage()
}

func sendExitStatus(usock *unixsocket.Usock, exitStatus string, err error) error {
	if err != nil {
		return err
	}

	_, err = usock.WriteMessage(exitStatus)
	return err
}

func receivePidFromCommand(commandUsock *unixsocket.Usock, err error) (int, error) {
	if err != nil {
		return -1, err
	}

	msg, err := commandUsock.ReadMessage()
	if err != nil {
		return -1, err
	}
	intPid, _, _ := messages.ParsePidMessage(msg)

	return intPid, err
}

func sendCommandPidToClient(usock *unixsocket.Usock, pid int, err error) error {
	if err != nil {
		return err
	}

	strPid := strconv.Itoa(pid)
	_, err = usock.WriteMessage(strPid)

	return err
}

func bootNewCommand(slaveNode *processtree.SlaveNode, command string, err error) (*unixsocket.Usock, error) {
	if err != nil {
		return nil, err
	}

	request := &processtree.CommandRequest{command, make(chan *os.File)}
	slaveNode.RequestCommandBoot(request)
	commandFile := <-request.Retchan // TODO: don't really want to wait indefinitely.
	// defer commandFile.Close() // TODO: can't do this here anymore.

	return unixsocket.NewUsockFromFile(commandFile)
}

func sendTTYToCommand(commandUsock *unixsocket.Usock, clientFile *os.File, err error) error {
	if err != nil {
		return err
	}

	return commandUsock.WriteFD(int(clientFile.Fd()))
}
