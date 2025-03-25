package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const socketPath = "/tmp/websocket.sock"

var (
	connections sync.Map // key: int (fd), value: net.Conn

	// Prometheus metric: number of currently active WebSocket connections
	activeConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "websocket_connections_active",
		Help: "Number of active WebSocket connections",
	})

	// Prometheus metric: total number of WebSocket connections handled
	totalConnections = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "websocket_connections_total",
		Help: "Total number of WebSocket connections handled",
	})

	// Prometheus metric: total number of messages sent to clients
	messagesSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "websocket_messages_sent_total",
		Help: "Total number of WebSocket messages sent to clients",
	})
)

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(activeConnections)
	prometheus.MustRegister(totalConnections)
	prometheus.MustRegister(messagesSent)
}

// setSocketGroup sets the group of the socket file to 'nobody' and makes it writable
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

// receiveFD receives a file descriptor from another process via Unix domain socket and SCM_RIGHTS
func receiveFD(socket *net.UnixConn) (int, error) {
	buf := make([]byte, 1)
	oob := make([]byte, syscall.CmsgSpace(4))

	_, _, _, _, err := socket.ReadMsgUnix(buf, oob)
	if err != nil {
		return -1, err
	}

	// Parse control message and extract file descriptor
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

// handleConnection manages a WebSocket connection: reads frames, sends replies, and tracks metrics
func handleConnection(conn net.Conn, fd int) {
	activeConnections.Inc()
	totalConnections.Inc()

	defer func() {
		conn.Close()
		connections.Delete(fd)
		activeConnections.Dec()
		log.Printf("Connection closed (fd=%d)", fd)
	}()

	log.Printf("New WebSocket connection (fd=%d)", fd)

	go func() {
		for {
			// Read WebSocket frames from the client
			messages, err := wsutil.ReadClientMessage(conn, nil)
			if err != nil {
				log.Printf("read error (fd=%d): %v", fd, err)
				return
			}

			// Handle different types of frames
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
		messagesSent.Inc()
		log.Printf("Message sent (fd=%d): %s", fd, msg)
	}
}

// startSCMRightsListener listens for incoming file descriptors on a Unix socket and serves them
func startSCMRightsListener() {
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

	log.Println("Listening for incoming file descriptors via Unix socket...")

	for {
		conn, err := unixListener.AcceptUnix()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}

		// Receive and convert the file descriptor to a net.Conn
		fd, err := receiveFD(conn)
		conn.Close()
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

// startNativeWebSocketServer serves native WebSocket connections and exposes /metrics for Prometheus
func startNativeWebSocketServer() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		fd := -1 // Native connections do not come with SCM_RIGHTS
		go handleConnection(conn, fd)
	})

	http.Handle("/metrics", promhttp.Handler())

	log.Println("Starting native WebSocket server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("WebSocket server error: %v", err)
	}
}

// main starts both the SCM_RIGHTS listener and native WebSocket server
func main() {
	go startSCMRightsListener()
	startNativeWebSocketServer()
}
