package main

import (
    "net"
    "fmt"
)

type A struct {
    Name string `json:"name"`
    Ttl int `json:"ttl"`
    Value string `json:"value"`
}

func getQuestionDomain(data []byte) (result string) {
    isLength := true
    var length int
    var currentLabel []byte
    var labels []string

    for i, b := range(data) {
        if (b == byte(0)) {
            break
        }

        if (isLength) {
            isLength = false
            length = int(b) + i
        } else {
            currentLabel = append(currentLabel, b)
            if (i == length) {
                isLength = true
                labels = append(labels, string(currentLabel))
                currentLabel = []byte{}
            }
        }
    }

    fmt.Println(labels)

    for _, label := range(labels) {
        result += label + "."
    }
    
    return
}

func getFlags(data []byte) []byte {
    
    var flags1 byte

    // QR
    flags1 |= byte(1 << 7)
    
    // OPCODE
    byte1 := data[0]
    for i := 1; i < 5; i++ {
        mask := byte(1 << uint(i))
        flags1 |= byte1 & mask
    }

    // AA
    flags1 |= byte(1 << 2)
    
    // TC
    flags1 |= byte(0 << 1)
    
    // RD
    flags1 |= byte(0)

    // Second bytes flag will be all 0, as we dont support recursion
    // or error handling yet :D
   
    flags2 := byte(0)

    return []byte{flags1, flags2}
}

func buildResponse(data []byte) []byte {
    id := data[:2]
    fmt.Printf("ID: %b\n", id)

    flags := getFlags(data[2:4])
    fmt.Printf("Flags: %b\n", flags)
    
    // QDCOUNT
    qCount := []byte{0, 1}
    fmt.Printf("QDCount: %b\n", qCount)


    questionDomain := getQuestionDomain(data[12:])
    fmt.Println(questionDomain)
    // ANCOUNT

    return data[:2]
}

func main() {
    conn, err := net.ListenUDP("udp", &net.UDPAddr{
        Port: 9000,
        IP: net.ParseIP("127.0.0.1"),
    })
    if err != nil {
        panic(err)
    }

    defer conn.Close()
    fmt.Printf("Server listening on %s\n", conn.LocalAddr().String())

    for {
        message := make([]byte, 512)
        rlen, remote, err := conn.ReadFromUDP(message[:])
        if err != nil {
            fmt.Println(err)
            panic(err)
        }

        data := message[:rlen]
        
        fmt.Printf("Query: %b\n", data)

        conn.WriteToUDP(buildResponse(data), remote)
    }
}

