package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
)


const (
    AType int = 1 
    NSType = 2
    CNameType = 5
    SOAType = 6
    MXType = 15
    TXTType = 16
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

func getQuestionDomain(data []byte) (result []string, questionType int) {
    isLength := true
    var length int
    var currentLabel []byte

    for i, b := range(data) {
        if (b == byte(0)) {
            length += 2
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

    questionTypeBytes := data[length:length + 2]
    questionType = int(questionTypeBytes[0]) + int(questionTypeBytes[1])
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
        name += label + "."
    }
    zones := getZones()
    zone, err := findZone(name, zones)
    if err != nil {
        panic("Zoone not found ")
    }
    return zone.A
}

func answerDatagram(answer A, cache *map[string]byte, responseLength int) (result []byte) {
    var nameBytes []byte
    var endsWithPointer bool
    localCache := *cache

    // NAME
    if (answer.Name == "@") {
        result = append([]byte{192, 12})
    } else {
        nameLables := strings.Split(answer.Name, ".")
        for _, label := range(nameLables) {

            if ref, in := localCache[label]; in {
                pointer := []byte{192, ref}
                nameBytes = append(nameBytes, pointer...)
                endsWithPointer = true
            } else {
                localCache[label] = byte(responseLength + len(nameBytes))
                nameBytes = append(nameBytes, byte(len(label)))
                for _, c := range(label) {
                    nameBytes = append(nameBytes, byte(c))
                }
                endsWithPointer = false
            }
        }
        if (!endsWithPointer) {
            nameBytes = append(nameBytes, []byte{0}...)
        }
    }
    
    *cache = localCache
    result = append(result, nameBytes...)

    // TYPE
    typeBytes := []byte{0, byte(AType)}
    result = append(result, typeBytes...)

    // CLASS
    classBytes := []byte{0, 1}
    result = append(result, classBytes...)

    // TTL
    ttlBytes := make([]byte, 4)
    binary.BigEndian.PutUint32(ttlBytes, uint32(answer.Ttl))
    result = append(result, ttlBytes...)

    // RDLENGTH
    lengthBytes := []byte{0, 4}
    result = append(result, lengthBytes...)

    // RDATA
    dataBytes := make([]byte, 4)
    for i, part := range(strings.Split(answer.Value, ".")) {
        partNum, err := strconv.Atoi(part)
        if err != nil {
            panic("Invalid IPv4 address")
        }
        dataBytes[i] = byte(partNum)
    }
    result = append(result, dataBytes...)

    return
}

func answersToBytes(answers []A, responseLength int) (result []byte) { 
    cache := make(map[string]byte)

    for _, answer := range(answers) {
        answerBytes := answerDatagram(answer, &cache, responseLength)
        responseLength += len(answerBytes)
        result = append(result, answerBytes...)
    }
    return
}

func getQuestion(questionDomain []string, questionType int) (result []byte) {
    for _, label := range(questionDomain) {
        labelLenght := byte(len(label))
        labelBytes := []byte{labelLenght}
        for _, c := range(label) {
            labelBytes = append(labelBytes, byte(c))
        }
        result = append(result, labelBytes...)
    }
    result = append(result, byte(0))

    // QTYPE
    qType := []byte{0, byte(questionType)}
    result = append(result, qType...)

    // QCLASS
    qClass := []byte{0, 1}
    result = append(result, qClass...)
    return
}



func buildResponse(data []byte) (result []byte) {
    id := data[:2]
    result = append(result, id...)

    flags := getFlags(data[2:4])
    result = append(result, flags...)
    
    // QDCOUNT
    qCount := []byte{0, 1}
    result = append(result, qCount...)

    // ANCOUNT
    questionDomain, questionType := getQuestionDomain(data[12:])
    records := getAnswers(questionDomain)
    anCount := make([]byte, 2)
    binary.BigEndian.PutUint16(anCount, uint16(len(records)))
    result = append(result, anCount...)

    nsCount := []byte{0, 0}
    result = append(result, nsCount...)

    arCount := []byte{0, 0}
    result = append(result, arCount...)

    question := getQuestion(questionDomain, questionType)
    result = append(result, question...)
    
    if (questionType == AType) {
        answers := answersToBytes(records, len(result))
        result = append(result, answers...)
    }

    return
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
        
        conn.WriteToUDP(buildResponse(data), remote)
    }
}

