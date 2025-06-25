package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	"net"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type MessageType struct {
	Name    string
	MsgType uint16
	MinSize int
	MaxSize int
}

var messageTypes = []MessageType{
	{"Ping", 1, 10, 20},
	{"Get", 2, 100, 500},
	{"Data", 3, 1000, 5000},
}

func main() {
	c := make(chan int16, 1000)
	for i := 0; i < 420; i++ {
		c <- int16(i)
	}

	log.Println("start")
	for {
		log.Println("outer loop")

	Loop:
		for i := 0; i < 100; i++ {
			select {
			case item := <-c:

				log.Println("processing: ", item)
			default:
				log.Println("breaking...")
				break Loop
			}
		}

		log.Println("sleeping...")
		time.Sleep(time.Millisecond * 50)

	}

	// var (
	// 	serverAddr     = flag.String("addr", "localhost:4411", "WebSocket server address")
	// 	numClients     = flag.Int("clients", 30000, "Number of clients to spawn")
	// 	minMsgFreqMs   = flag.Int("minfreq", 5000, "Minimum milliseconds between client messages")
	// 	maxMsgFreqMs   = flag.Int("maxfreq", 20000, "Maximum milliseconds between client messages")
	// 	disconnectRate = flag.Float64("disrate", 0.00, "Chance per message interval to disconnect")
	// 	reconnectDelay = flag.Int("reconndelay", 3000, "Milliseconds before reconnecting after disconnect")
	// 	runDuration    = flag.Int("duration", 15, "Test run duration in seconds")
	// )
	// flag.Parse()

	// if *minMsgFreqMs > *maxMsgFreqMs {
	// 	log.Fatalf("minfreq cannot be greater than maxfreq")
	// }

	// log.Printf("Starting tester with %d clients, msg freq between %dms and %dms, disconnect rate %.2f, reconnect delay %dms",
	// 	*numClients, *minMsgFreqMs, *maxMsgFreqMs, *disconnectRate, *reconnectDelay)

	// ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*runDuration)*time.Second)
	// defer cancel()

	// wg := &sync.WaitGroup{}
	// for i := 0; i < *numClients; i++ {
	// 	wg.Add(1)
	// 	go func(clientID int) {
	// 		defer wg.Done()
	// 		clientLoop(ctx, *serverAddr, clientID, *minMsgFreqMs, *maxMsgFreqMs, *disconnectRate, *reconnectDelay)
	// 	}(i)
	// }

	// wg.Wait()
	// log.Println("Tester finished.")
}

func clientLoop(ctx context.Context, addr string, clientID, minFreqMs, maxFreqMs int, disRate float64, reconDelayMs int) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(clientID)))

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, _, _, err := ws.Dial(context.Background(), "ws://"+addr)
		if err != nil {
			log.Printf("[Client %d] Connection error: %v", clientID, err)
			time.Sleep(time.Second)
			continue
		}
		log.Printf("[Client %d] Connected", clientID)

		runClient(ctx, conn, clientID, minFreqMs, maxFreqMs, disRate, reconDelayMs, rnd)

		// Wait before reconnecting
		time.Sleep(time.Duration(reconDelayMs) * time.Millisecond)
	}
}

func runClient(ctx context.Context, conn net.Conn, clientID, minFreqMs, maxFreqMs int, disRate float64, reconDelayMs int, rnd *rand.Rand) {
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[Client %d] Context done, closing", clientID)
			return
		default:
		}

		// Random send interval for this message
		interval := time.Duration(rnd.Intn(maxFreqMs-minFreqMs+1)+minFreqMs) * time.Millisecond
		time.Sleep(interval)

		// Random disconnect chance
		if rnd.Float64() < disRate {
			log.Printf("[Client %d] Disconnecting (random)", clientID)
			return
		}

		// Pick random message type
		msgType := messageTypes[rnd.Intn(len(messageTypes))]

		// Pick random payload size in range
		size := rnd.Intn(msgType.MaxSize-msgType.MinSize+1) + msgType.MinSize

		msg := make([]byte, size)
		rnd.Read(msg)

		err := writeWSMessage(conn, msgType.MsgType, msg)
		if err != nil {
			log.Printf("[Client %d] Write error: %v", clientID, err)
			return
		}

		//log.Printf("[Client %d] Sent %s message (%d bytes)", clientID, msgType.Name, size)
	}
}

func writeWSMessage(conn net.Conn, msgType uint16, payload []byte) error {
	// Compose message header (6 bytes: 2 bytes msgType + 4 bytes length)
	header := make([]byte, 2)
	header[0] = byte(msgType >> 8)
	header[1] = byte(msgType & 0xff)

	// Concatenate header + payload into one slice
	msg := append(header, payload...)

	// Send as one WebSocket binary frame
	return wsutil.WriteClientBinary(conn, msg)
}
