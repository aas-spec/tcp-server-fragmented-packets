package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

var serverAddr = flag.String("server_addr", "localhost:5555", "server address")
var maxClients = flag.Int("max_clients", 10, "number of clients")

type ClientConnection struct {
	conn net.Conn
	tag  string
}

func main() {
	flag.Parse()
	go startServer(*serverAddr)
	clientChan := make([]chan string, 0)

	for i := 0; i < *maxClients; i++ {
		clientTag := strconv.Itoa(i + 1)
		clientChan = append(clientChan, make(chan string))
		go startClient(*serverAddr, clientTag, clientChan[i])
	}
	log.Println("Wait a few seconds...")
	time.Sleep(5 * time.Second)

	// send messages
	for i := 0; i < *maxClients; i++ {
		clientTag := strconv.Itoa(i + 1)
		msg := "Broadcast from client " + clientTag
		if i == 0 {
			msg = "2:Message to client 2 from " + clientTag
		}
		clientChan[i] <- msg
	}

	select {}
}

func startServer(addr string) {
	log.Printf("starting test tcp server: %s", addr)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("unable to start  server %s", err)
	}
	defer l.Close()

	var connMap = &sync.Map{}

	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		clientId := uuid.New().String()
		go handleClientConnection(conn, clientId, connMap)
	}
}

func startClient(serverAddr string, clientTag string, writeCh chan string) {
	c, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Fatalln(err)
	}
	defer c.Close()
	// send clientTag
	err = writeMessage(c, clientTag)
	if err != nil {
		log.Printf("tcp client: %s, unable to send tag: %s ", clientTag, err)
		return
	}

	readCh := make(chan string)
	errorCh := make(chan error)

	// Start reading
	go func(ch chan string, eCh chan error) {
		conBuf := bufio.NewReader(c)
		for {
			s, err := readMessage(conBuf)
			if err != nil {
				eCh <- err
				return
			}
			ch <- s
		}
	}(readCh, errorCh)

	ticker := time.Tick(time.Second)
	for {
		select {
		case readData := <-readCh:
			log.Printf("rx: tcp client: %s : %s", clientTag, readData)
		case err := <-errorCh:
			log.Printf("tcp client: %s, unable to read: %s ", clientTag, err)
			return
		case writeData := <-writeCh:
			log.Printf("tx: tcp client: %s : %s", clientTag, writeData)
			err = writeMessage(c, writeData)
			if err != nil {
				log.Printf("tcp client: %s, unable to send message: %s ", clientTag, err)
				return
			}
		case <-ticker:
		}
	}
}

func bytesToInt(b []byte) int {
	return int(binary.BigEndian.Uint16(b))
}

func intToBytes(i int) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(i))
	return b
}

func writeMessage(c net.Conn, msg string) error {
	// send len
	lenBytes := intToBytes(len(msg))

	_, err := c.Write(lenBytes)
	if err != nil {
		return err
	}
	data := []byte (msg)
	res, err := c.Write(data)
	if err != nil {
		return err
	}
	if res != len(msg) {
		errMsg := fmt.Sprintf("res != msglen: %d", res)
		return errors.New(errMsg)
	}
	return nil
}

func readMessage(r *bufio.Reader) (string, error) {
	lenBytes := make([]byte, 0)
	for i := 0; i < 2; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		lenBytes = append(lenBytes, b)
	}
	lenData := bytesToInt(lenBytes)

	// reading data
	data := make([]byte, 0)
	for i := 0; i < int(lenData); i++ {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		data = append(data, b)
	}
	return string(data), nil
}

func handleClientConnection(c net.Conn, clientId string, connMap *sync.Map) {
	defer func() {
		c.Close()
		connMap.Delete(clientId)
	}()
	conBuf := bufio.NewReader(c)
	clientTag, err := readMessage(conBuf)
	if err != nil {
		log.Printf("client %s: unable to read: %s", clientId, err)
		return
	}
	// store client item
	clientConn := ClientConnection{
		conn: c,
		tag:  clientTag,
	}
	connMap.Store(clientId, clientConn)

	// reading data
	for {
		// reading length
		clientMessage, err := readMessage(conBuf)
		if err != nil {
			log.Printf("client %s %s: unable to read: %s", clientId, clientTag, err)
			return
		}

		msg := strings.Split(clientMessage, ":")
		targetClient := ""
		if len(msg) > 1 {
			clientMessage = msg[1]
			targetClient = msg[0]
		}
		// send messages
		connMap.Range(func(key, value interface{}) bool {
			clientItem, _ := value.(ClientConnection)
			if (targetClient == "" && clientItem.tag != clientTag) || targetClient == clientItem.tag {
			err := writeMessage(clientItem.conn, clientMessage)
				if err != nil {
					log.Printf("client %s: unable to write: %v", clientItem.tag, err)
				}
			}
			return true
		})
	}
}
