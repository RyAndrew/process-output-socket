package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
)

var firstConnection = true
var mutex = &sync.Mutex{}

var socketCloseChannels []chan bool
var socketDataChannels []chan []byte

var stdOutCh = make(chan []byte, 512)
var stdErrCh = make(chan []byte, 512)
var socketChan = make(chan []byte, 1024)
var processDoneChan = make(chan bool)

func main() {
	go acceptConnections()
	for {
		select {
		case <-processDoneChan:
			fmt.Println("Process Done")
			for _, ch := range socketCloseChannels {
				ch <- true
			}
		case outByteArr := <-stdOutCh:
			for _, ch := range socketDataChannels {
				ch <- outByteArr
			}
		case errByteArr := <-stdErrCh:
			for _, ch := range socketDataChannels {
				ch <- errByteArr
			}
		}
	}
}
func acceptConnections() {
	port := "25560"

	tcpAddr, err := net.ResolveTCPAddr("tcp4", ":"+port)
	checkFatalError(err)

	fmt.Println("Listening on port " + port)

	listener, err := net.ListenTCP("tcp4", tcpAddr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {

	mutex.Lock()
	if firstConnection {
		firstConnection = false
		mutex.Unlock()
		go launchProcess()
	} else {
		mutex.Unlock()
	}
	var sockCloseChan = make(chan bool)
	var sockDataChan = make(chan []byte, 512)

	socketCloseChannels = append(socketCloseChannels, sockCloseChan)
	socketDataChannels = append(socketDataChannels, sockDataChan)

	for {
		select {
		case <-sockCloseChan:
			conn.Write([]byte("Process Done!\r\n"))
			conn.Close()
		case msg := <-sockDataChan:
			conn.Write(msg)
		}
	}

}

type stdOutChannel struct {
	//*sync.Mutex
}

func newStdOutChannel() *stdOutChannel {
	return &stdOutChannel{
	//Mutex: &sync.Mutex{},
	}
}

// io.Writer interface is only this method
func (fd *stdOutChannel) Write(p []byte) (int, error) {
	//fd.Lock()
	//defer fd.Unlock()
	fmt.Printf("stdout read %d bytes\n", len(p))
	fmt.Printf("%s", p)

	stdOutCh <- p

	return len(p), nil
}

type stdErrChannel struct {
	//*sync.Mutex
}

func newStdErrChannel() *stdErrChannel {
	return &stdErrChannel{
	//Mutex: &sync.Mutex{},
	}
}

// io.Writer interface is only this method
func (fd *stdErrChannel) Write(p []byte) (int, error) {
	//fd.Lock()
	//defer fd.Unlock()

	fmt.Printf("stderr read %d bytesn\n", len(p))
	fmt.Printf("%s", p)

	stdErrCh <- p
	return len(p), nil
}

func launchProcess() {
	var err error
	fmt.Println("Launching process")

	cmd := exec.Command("/home/miner/gocode/src/github.com/ryandrew/stdout-once-per-sec/stdout-once-per-sec")

	stdoutWriter := newStdOutChannel()
	stderrWriter := newStdErrChannel()
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	err = cmd.Start()
	checkFatalError(err)

	err = cmd.Wait()
	checkFatalError(err)

	processDoneChan <- true
}

func checkFatalError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
