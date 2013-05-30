package main

import (
    "net"
    "io"
    "encoding/json"
    "encoding/binary"
    "compress/gzip"
    "fmt"

    "github.com/yjh0502/gb"
)

type cBench struct {
    c net.Conn
    reader *gzip.Reader
    writer *gzip.Writer

    count int
}

type UsersGet struct {
    Req string `json:"req"`
    Id string `json:"id"`
}

func (b *cBench) Execute() (done bool, err error) {
    b.count++
    if b.count > 1 {
        return true, nil
    }

    data, err := json.Marshal(UsersGet{"/v1/users_get", "asdf"})
    if err != nil {
        return false, fmt.Errorf("Failed to marshal json: %s", err.Error())
    }

    if err = binary.Write(b.writer, binary.BigEndian, uint32(len(data))); err != nil {
        return false, fmt.Errorf("Failed to write length: %s", err.Error())
    }

    write_len, err := b.writer.Write(data)
    if write_len != len(data) {
        return false, fmt.Errorf("Data not all written: %d != %d", write_len, len(data))
    }
    if err != nil {
        return false, fmt.Errorf("Failed to write json: %s", err.Error())
    }
    b.writer.Flush()

    if b.reader == nil {
        if b.reader, err = gzip.NewReader(b.c); err != nil {
            return false, fmt.Errorf("Failed to create gzip reader: %s", err)
        }
    }

    var read_len uint32
    binary.Read(b.reader, binary.BigEndian, &read_len)

    readData := make([]byte, read_len)
    if _, err = io.ReadFull(b.reader, readData); err != nil {
        return false, fmt.Errorf("Failed to read all: %s", err.Error())
    }

    return false, nil
}

func benchInit() (gb.BenchmarkRunner, error) {
    var err error
    b := new(cBench)

    conn, err := net.Dial("tcp", "localhost:50001")
    if err != nil {
        return nil, fmt.Errorf("Failed to connect: %s\n", err)
    }

    b.c = conn
    b.writer = gzip.NewWriter(conn)

    return b, nil
}

func main() {
	b := gb.NewBench()
	b.Run(benchInit)
}
