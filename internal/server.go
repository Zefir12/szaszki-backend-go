package internal

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

func ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	log.Println("WebSocket server started on", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	_, err := ws.Upgrade(conn)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		conn.Close()
		return
	}
	defer conn.Close()

	br := wsutil.NewReader(conn, ws.StateServerSide)

	client := &ClientConn{
		Conn:   conn,
		UserID: 0,
	}
	defer func() {
		if client.UserID != 0 {
			RemoveClient(client.UserID)
		}
	}()

	done := make(chan struct{})

	// Ping sender goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := WriteMsg(conn, ServerCmds.Ping, nil)
				if err != nil {
					log.Println("Ping error:", err)
					close(done)
					return
				}
			case <-done:
				return
			}
		}
	}()

	handlePackets(conn, br, client)

}

func handlePackets(conn net.Conn, br *wsutil.Reader, client *ClientConn) {
	for {
		hdr, err := br.NextFrame()
		if err != nil {
			if err == io.EOF {
				return
			}
			if strings.Contains(err.Error(), "wsarecv") {
				return
			}
			log.Println("Frame read error:", err)
			return
		}

		// Only handle binary frames
		if hdr.OpCode != ws.OpBinary {
			// Discard unwanted frame by reading and discarding data
			if _, err := io.CopyN(io.Discard, br, int64(hdr.Length)); err != nil {
				log.Println("Discard error:", err)
				return
			}
			continue
		}

		size := int(hdr.Length)
		bufPtr := GetBufferForSize(size)
		buf := *bufPtr
		if size > cap(buf) {
			log.Printf("Frame too large: %d bytes", size)
			PutBuffer(bufPtr)
			return
		}

		buf = buf[:size]
		_, err = io.ReadFull(br, buf)
		if err != nil {
			log.Println("Payload read error:", err)
			PutBuffer(bufPtr)
			return
		}

		if len(buf) < 2 {
			log.Println("Payload too short for MsgType")
			PutBuffer(bufPtr)
			continue
		}

		msgType := MsgType(binary.BigEndian.Uint16(buf[0:2]))
		payload := buf[2:]

		handleMessage(conn, msgType, payload, client)

		PutBuffer(bufPtr)
	}
}

// func handleConn(conn net.Conn) {
// 	_, err := ws.Upgrade(conn)
// 	if err != nil {
// 		log.Println("WebSocket upgrade error:", err)
// 		conn.Close()
// 		return
// 	}
// 	defer conn.Close()
// 	buf := make([]byte, 6000) // max buffer size
// 	for {

// 		n, err := conn.Read(buf)
// 		if err != nil {
// 			if err == io.EOF {
// 				return
// 			}
// 			if strings.Contains(err.Error(), "wsarecv") {
// 				return
// 			}
// 			log.Println("Frame read error:", err)
// 		}

// 		if n < 2 {
// 			log.Println("Received too few bytes to parse MsgType")
// 			continue
// 		}

// 		msgType := MsgType(binary.BigEndian.Uint16(buf[0:2]))
// 		payload := buf[2:n]

// 		handleMessage(conn, msgType, payload)
// 	}
// }
