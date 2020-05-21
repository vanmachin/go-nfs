package main

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"net"
)

func main() {
	println("Opening socket")
	ln, err := net.Listen("tcp", ":2049")
	if err != nil {
		panic(err)
	}
	println("Socket opened")

	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go handleConnection(conn)

	}
}

var nfsProgramNumber uint32 = 100003

func handleConnection(conn net.Conn) {
	defer conn.Close()

	for {
		bufpayload := getRPCRecord(conn)
		if bufpayload == nil {
			return
		}

		print(hex.Dump(bufpayload[0:]))

		if binary.BigEndian.Uint32(bufpayload[16:20]) != nfsProgramNumber {
			panic("not nfsv")
		}
		if binary.BigEndian.Uint32(bufpayload[20:24]) != 4 {
			panic("not nfsv4")
		}

		procedure := binary.BigEndian.Uint32(bufpayload[24:28])
		switch procedure {
		case 0:
			// NULL
			resp := make([]byte, 0)
			resp = append(resp, []byte{0, 0, 0, 0}...)             // fragment header
			resp = append(resp, bufpayload[4:8]...)                // xid
			resp = append(resp, []byte{0, 0, 0, 1}...)             // response
			resp = append(resp, []byte{0, 0, 0, 0}...)             // accepted
			resp = append(resp, []byte{0, 0, 0, 0, 0, 0, 0, 0}...) // auth
			resp = append(resp, []byte{0, 0, 0, 0}...)             // accept state

			putRPCRecord(conn, resp)

		case 1:
			// COMPOUND
		default:
			panic("unknown procedure")
		}

		return
	}
}

func getRPCRecord(conn net.Conn) []byte {

	bufheader := make([]byte, 4)

	nb, err := conn.Read(bufheader)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		panic(err)
	}
	if nb != 4 {
		panic("header too short ")
	}
	bufheader[0] = bufheader[0] & 0x7F // TODO Handle last fragment == false
	payloadSize := binary.BigEndian.Uint32(bufheader)
	bufpayload := make([]byte, payloadSize+4)
	copy(bufpayload, bufheader)
	nb, err = conn.Read(bufpayload[4:])
	if err != nil {
		if err == io.EOF {
			return nil
		}
		panic(err)
	}
	if uint32(nb) != payloadSize {
		panic("not enough payload ")
	}
	return bufpayload
}

func putRPCRecord(conn net.Conn, buffer []byte) {
	binary.BigEndian.PutUint32(buffer[0:4], uint32(len(buffer)-4))
	buffer[0] = buffer[0] | 0x80
	nb, err := conn.Write(buffer[0:])
	if err != nil {
		panic(err)
	}
	if nb != len(buffer) {
		panic("couldn't write all bytes")
	}
}

func getNFSv4Compound(buffer []byte) compound {

	var ret compound

	tagLength := 0
	for i := 0; i < len(buffer); i++ {
		if buffer[i] == 0 {
			tagLength = i + 1
		}
	}
	ret.tag = string(buffer[0:tagLength])
	ret.minor = binary.BigEndian.Uint32(buffer[tagLength : tagLength+4])

	return ret
}

type operation struct {
	opcode uint32
}

type compound struct {
	tag        string
	minor      uint32
	operations []operation
}
