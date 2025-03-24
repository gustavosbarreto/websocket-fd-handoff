package main

import (
	"log"
	"net"
	"os"
	"os/user"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

const socketPath = "/tmp/websocket.sock"

// Concurrent map to store active WebSocket connections
var connections sync.Map // key: int (fd), value: net.Conn

func setSocketGroup(path string) {
	nobody, err := user.Lookup("nobody")
	if err != nil {
		log.Fatalf("failed to lookup user 'nobody': %v", err)
	}

	gid, err := strconv.Atoi(nobody.Gid)
	if err != nil {
		log.Fatalf("invalid GID for nobody: %v", err)
	}

	if err := os.Chown(path, -1, gid); err != nil {
		log.Fatalf("failed to change group of socket: %v", err)
	}

	if err := os.Chmod(path, 0660); err != nil {
		log.Fatalf("failed to set permissions on socket: %v", err)
	}
}

// Receives a file descriptor sent via SCM_RIGHTS
func receiveFD(socket *net.UnixConn) (int, error) {
	buf := make([]byte, 1)
	oob := make([]byte, syscall.CmsgSpace(4))

	_, _, _, _, err := socket.ReadMsgUnix(buf, oob)
	if err != nil {
		return -1, err
	}

	msgs, err := syscall.ParseSocketControlMessage(oob)
	if err != nil {
		return -1, err
	}

	if len(msgs) == 0 {
		return -1, syscall.EINVAL
	}

	fds, err := syscall.ParseUnixRights(&msgs[0])
	if err != nil {
		return -1, err
	}

	if len(fds) == 0 {
		return -1, syscall.EINVAL
	}

	return fds[0], nil
}

func handleConnection(conn net.Conn, fd int) {
	defer func() {
		conn.Close()
		connections.Delete(fd)
		log.Printf("Connection closed (fd=%d)", fd)
	}()

	log.Printf("New WebSocket connection (fd=%d)", fd)

	// Read frames from client
	go func() {
		for {
			messages, err := wsutil.ReadClientMessage(conn, nil)
			if err != nil {
				log.Printf("read error (fd=%d): %v", fd, err)
				return
			}

			for _, frame := range messages {
				switch frame.OpCode {
				case ws.OpText:
					log.Printf("Text message (fd=%d): %s", fd, string(frame.Payload))

				case ws.OpPing:
					log.Printf("PING received (fd=%d), sending PONG", fd)
					if err := ws.WriteFrame(conn, ws.NewPongFrame(nil)); err != nil {
						log.Printf("PONG send error (fd=%d): %v", fd, err)
						return
					}

				case ws.OpClose:
					log.Printf("Client closed connection (fd=%d)", fd)
					return
				}
			}
		}
	}()

	// Periodically send messages to client
	for {
		time.Sleep(5 * time.Second)
		msg := "Hello from backend"
		if err := wsutil.WriteServerText(conn, []byte(msg)); err != nil {
			log.Printf("write error (fd=%d): %v", fd, err)
			return
		}
		log.Printf("Message sent (fd=%d): %s", fd, msg)
	}
}

func main() {
	os.Remove(socketPath)

	unixAddr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		log.Fatal("resolve error:", err)
	}

	unixListener, err := net.ListenUnix("unix", unixAddr)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer unixListener.Close()

	setSocketGroup(socketPath)

	log.Println("Listening for incoming file descriptors...")

	for {
		conn, err := unixListener.AcceptUnix()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}

		fd, err := receiveFD(conn)
		conn.Close() // done with Unix connection
		if err != nil {
			log.Println("fd receive error:", err)
			continue
		}

		log.Printf("FD received: %d", fd)

		file := os.NewFile(uintptr(fd), "websocket_fd")
		tcpConn, err := net.FileConn(file)
		file.Close()
		if err != nil {
			log.Printf("failed to create net.Conn (fd=%d): %v", fd, err)
			continue
		}

		connections.Store(fd, tcpConn)
		go handleConnection(tcpConn, fd)
	}
}
