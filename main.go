package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
)

type A struct {
    Name string `json:"name"`
    Ttl int `json:"ttl"`
    Value string `json:"value"`
}

type NS struct {
    Host string `json:"host"`
}

type SOA struct {
    MName string `json:"mname"`
    RName string `json:"rname"`
    Serial string `json:"serial"`
    Refresh int `json:"refresh"`
    Retry int `json:"retry"`
    Expire int `json:"expire"`
    Minimum int `json:"minimum"`
}

type Zone struct {
    Origin string `json:"$origin"`
    TTL int `json:"$ttl"`
    SOA SOA `json:"soa"`
    NS []NS `json:"ns"`
    A []A `json:"a"`
}

func getZones() (result []Zone) {
    files, err := ioutil.ReadDir("./zones")
    if err != nil {
        panic("Zone dir not found")
    }

    for _, file := range(files) {
        var zone Zone
        fmt.Println(file.Name())
        content, err := os.ReadFile("./zones/" + file.Name())
        if err != nil {
            fmt.Println(err)
            panic("Fatalnie")
        }
        json.Unmarshal(content, &zone)
        result = append(result, zone)
    }

    return
}

func findZone(origin string, zones []Zone) (Zone, error) {
    for _, zone := range(zones) {
        if zone.Origin == origin {
            return zone, nil
        }
    }
    return Zone{}, errors.New("Zone not found")
}

func getQuestionDomain(data []byte) (result []string) {
    isLength := true
    var length int
    var currentLabel []byte

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
                result = append(result, string(currentLabel))
                currentLabel = []byte{}
            }
        }
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

func getAnswers(questionDomain []string) []A {
    var name string
    for _, label := range(questionDomain) {
        name += label
    }
    zones := getZones()
    fmt.Println(zones)
    zone, err := findZone(name, zones)
    if err != nil {
        panic("Zoone not found ")
    }
    return zone.A
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
    
    answers := getAnswers(questionDomain)
    fmt.Println(answers)
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

