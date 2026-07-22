package main

import (
	"flag"
	"fmt"
	"net"
	"os"
)

func main() {
	serverMode := flag.Bool("s", false, "run echo server mode")
	clientMode := flag.Bool("c", false, "run echo client mode")
	addr := flag.String("addr", ":8080", "address to listen on or connect to")
	flag.Parse()

	switch {
	case *clientMode:
		runClient(*addr)
	case *serverMode:
		runServer(*addr)
	default:
		runServer(*addr)
	}
}

func runServer(addr string) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen error: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("echo server listening on %s\n", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "accept error: %v\n", err)
			continue
		}
		go handleConn(conn)
	}
}

func runClient(addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	query(conn)

}

func handleConn(conn net.Conn) {
	defer conn.Close()

	fmt.Fprintln(conn, "welcome to tcp echo server")
	for {
		if err := handleEcho(conn); err != nil {
			fmt.Fprintf(os.Stderr, "echo error: %v\n", err)
			return
		}
	}
}
