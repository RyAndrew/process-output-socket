package main

import (
	"bufio"
	"container/list"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var startProcessOnConnect = true
var mutex = &sync.Mutex{}

type conChanListElement struct {
	Connection *net.TCPConn
}

var activeConnectionsList = list.New()

var processOutputChannel = make(chan []byte, 512)
var dataFromSocketChannel = make(chan []byte, 1024)

var processDoneChan = make(chan bool)

func main() {
	go acceptConnections()

	for {
		select {
		case fromSocketBytes := <-dataFromSocketChannel:
			sendDataToAllConnections(fromSocketBytes)
		case <-processDoneChan:
			fmt.Println("Process Done")
			sendDataToAllConnections([]byte("Process Done\n"))

			mutex.Lock()
			startProcessOnConnect = true
			mutex.Unlock()
		case outByteArr := <-processOutputChannel:
			sendDataToAllConnections(outByteArr)
		}
	}
}
func acceptConnections() {
	port := "25560"

	tcpAddr, err := net.ResolveTCPAddr("tcp4", ":"+port)
	checkFatalError(err)

	listener, err := net.ListenTCP("tcp4", tcpAddr)
	if err != nil {
		fmt.Println("Fatal Error #1 - cannot bind to port", port)
		fmt.Println(err)
		os.Exit(3)
	}

	fmt.Println("Listening on port " + port)

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn *net.TCPConn) {

	fmt.Printf("Client connected: %s\n", conn.RemoteAddr())

	if !initializeClient(conn) {
		return
	}

	mutex.Lock()
	if startProcessOnConnect {
		startProcessOnConnect = false
		mutex.Unlock()
		go launchProcess()
	} else {
		mutex.Unlock()
	}

	mutex.Lock()
	connectionListEntry := activeConnectionsList.PushBack(conChanListElement{conn})
	mutex.Unlock()

	go readFromConnection(conn, connectionListEntry)

}
func initializeClient(conn *net.TCPConn) bool {

	challengeResponseSucceeded := make(chan bool)
	go initializeReadChallengeResponse(conn, challengeResponseSucceeded)

	timeoutTimer := time.NewTimer(time.Second * 4)

	select {
	case <-challengeResponseSucceeded:
		timeoutTimer.Stop()
		return true
	case <-timeoutTimer.C:
		fmt.Println("Connection init failed, timeout exceeded")
		conn.Close()
		return false
	}

}
func initializeReadChallengeResponse(conn *net.TCPConn, success chan bool) {
	conn.Write([]byte("PI"))

	scanner := bufio.NewScanner(conn)
	//for scanner.Scan() {
	scanner.Scan()

	fmt.Println("Scan...")
	//challengeResponse := scanner.Bytes()
	challengeResponse := scanner.Text()
	fmt.Printf("INIT Read %d bytes from socket %s\n", len(challengeResponse), conn.RemoteAddr())
	fmt.Println(challengeResponse)
	//fmt.Println(string(challengeResponse))
	//if strings.Compare(string(challengeResponse), "ZZ") != 0 {
	if strings.Compare(challengeResponse, "ZZ") != 0 {
		fmt.Println("Connection init failed, invalid challenge response")
		conn.Close()
		return
	}
	conn.Write([]byte("PIZZA!\n"))
	success <- true
	return
	//}
}
func readFromConnection(conn *net.TCPConn, chanListEntry *list.Element) {

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		socketData := scanner.Bytes()
		fmt.Printf("Read %d bytes from socket %s\n", len(socketData), conn.RemoteAddr())

		withNewLine := append(socketData, '\n')
		fmt.Printf("%s", withNewLine)
		dataFromSocketChannel <- withNewLine
	}
	err := scanner.Err()
	if err != nil {
		fmt.Println("Socket Scanner Error: ", err)
	}
	fmt.Println("Client disconnect: ", conn.RemoteAddr())

	mutex.Lock()
	activeConnectionsList.Remove(chanListEntry)
	mutex.Unlock()

}
func sendDataToAllConnections(data []byte) {
	i := 0
	for e := activeConnectionsList.Front(); e != nil; e = e.Next() {
		i++
		fmt.Println("Sending Data to ", i)
		el := e.Value.(conChanListElement)
		el.Connection.Write(data)
	}
}

type stdOutChannel struct {
}

func (fd *stdOutChannel) Write(p []byte) (int, error) {
	fmt.Printf("stdout read %d bytes\n", len(p))
	fmt.Printf("%s", p)

	processOutputChannel <- p

	return len(p), nil
}

type stdErrChannel struct {
}

func (fd *stdErrChannel) Write(p []byte) (int, error) {

	fmt.Printf("stderr read %d bytesn\n", len(p))
	fmt.Printf("%s", p)

	processOutputChannel <- p
	return len(p), nil
}

func launchProcess() {
	var err error
	fmt.Println("Launching process")

	cmd := exec.Command(configCommand, "4")
	//cmd := exec.Command("java", "-jar", "spigot-1.12.2.jar")
	//cmd.Dir = "/home/miner/mineframe/minecraft"

	cmd.Stdout = new(stdOutChannel)
	cmd.Stderr = new(stdErrChannel)

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
