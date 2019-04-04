package main

import (
    "io"
    "log"
    "net"
)

func forward(conn net.Conn, target string) {
    client, err := net.Dial("tcp", target)
    if err != nil {
        log.Fatalf("failed to dial: %v", err)
    }
    log.Printf("connected: %v\n", conn)
    go func() {
        defer client.Close()
        defer conn.Close()
        io.Copy(client, conn)
    }()
    go func() {
        defer client.Close()
        defer conn.Close()
        io.Copy(conn, client)
    }()
}

func serve(listen, target string) {
    listener, err := net.Listen("tcp", listen)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Fatalf("failed to accept: %v", err)
        }
		log.Printf("accepted: %v\n", conn)

        go forward(conn, target)
    }
}