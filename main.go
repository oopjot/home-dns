package main

import (
    "net"
    "fmt"
)

func handleConnection(conn net.Conn) {
    fmt.Println(conn)
}

func main() {
    ln, err := net.Listen("tcp", ":53")

    if err != nil {
        fmt.Println("Connection error ", err)
    }

    for {
        conn, err := ln.Accept()
        if err != nil {
            fmt.Println("This is an error: ", err)
        }
        go handleConnection(conn)
    }
}

