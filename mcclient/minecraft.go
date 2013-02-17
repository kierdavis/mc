package mcclient

import (
	"crypto/rsa"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const Tick = time.Second / 20

type Kick string

func (kick Kick) Error() (s string) {
	return "Kicked: " + string(kick)
}

type Signal chan struct{}
type Metadata map[uint8]interface{}

type Slot struct {
	ID     int16
	Count  int8
	Damage int16
	Data   []byte
}

type Position struct {
	X int32
	Y int32
	Z int32
}

type ExplosionRecord struct {
	X int8
	Y int8
	Z int8
}

type LogReadWriter struct {
	inner io.ReadWriter
}

func (lrw LogReadWriter) Read(data []byte) (n int, err error) {
	n, err = lrw.inner.Read(data)
	fmt.Printf("R %X\n", data[:n])
	return n, err
}

func (lrw LogReadWriter) Write(data []byte) (n int, err error) {
	fmt.Printf("W %X\n", data)
	return lrw.inner.Write(data)
}

type Client struct {
	ErrChan       chan error
	DebugWriter   io.Writer
	PacketLogging bool
	HandleMessage func(string)
	StoreWorld    bool
	Columns       map[ColumnCoord]*Column

	PlayerX        float64
	PlayerY        float64
	PlayerZ        float64
	PlayerStance   float64
	PlayerYaw      float32
	PlayerPitch    float32
	PlayerOnGround bool

	netConn net.Conn
	conn    io.ReadWriter

	stopHTTPKeepAlive  Signal
	stopPositionSender Signal

	username              string
	sessionId             string
	serverId              string
	serverAddr            string
	serverKeyMessage      []byte
	serverKey             *rsa.PublicKey
	serverVerifyToken     []byte
	encryptedVerifyToken  []byte
	sharedSecret          []byte
	encryptedSharedSecret []byte

	entityID   int32
	levelType  string
	serverMode int32
	dimension  int32
	difficulty int8
	maxPlayers uint8
}

func newClient(username string, sessionId string, debugWriter io.Writer) (client *Client) {
	client = &Client{
		ErrChan:            make(chan error),
		DebugWriter:        debugWriter,
		PacketLogging:      false,
		Columns:            make(map[ColumnCoord]*Column),
		stopHTTPKeepAlive:  make(Signal),
		stopPositionSender: make(Signal),
		username:           username,
		sessionId:          sessionId,
	}

	go client.HTTPKeepAlive()

	return client
}

// Removes Minecraft colour escapes.
func NoEscapes(input string) (output string) {
	start := 0

	for {
		end := strings.Index(input[start:], "\xC2\xA7")
		if end < 0 {
			break
		}

		end += start
		output += input[start:end]
		start = end + 3
	}

	output += input[start:]
	return output
}

// Converts Minecraft colour escapes to ANSI escape codes for printing in a terminal.
func ANSIEscapes(input string) (output string) {
	start := 0

	for {
		end := strings.Index(input[start:], "\xC2\xA7")
		if end < 0 {
			break
		}

		end += start
		output += input[start:end]

		switch input[end+2] {
		case '0':
			output += "\x1b[21m\x1b[30m"
		case '1':
			output += "\x1b[21m\x1b[34m"
		case '2':
			output += "\x1b[21m\x1b[32m"
		case '3':
			output += "\x1b[21m\x1b[36m"
		case '4':
			output += "\x1b[21m\x1b[31m"
		case '5':
			output += "\x1b[21m\x1b[35m"
		case '6':
			output += "\x1b[21m\x1b[33m"
		case '7':
			output += "\x1b[21m\x1b[37m"
		case '8':
			output += "\x1b[1m\x1b[30m"
		case '9':
			output += "\x1b[1m\x1b[34m"
		case 'a', 'A':
			output += "\x1b[1m\x1b[32m"
		case 'b', 'B':
			output += "\x1b[1m\x1b[36m"
		case 'c', 'C':
			output += "\x1b[1m\x1b[31m"
		case 'd', 'D':
			output += "\x1b[1m\x1b[35m"
		case 'e', 'E':
			output += "\x1b[1m\x1b[33m"
		case 'f', 'F':
			output += "\x1b[1m\x1b[37m"
		}

		start = end + 3
	}

	output += input[start:] + "\x1b[21m\x1b[39m"

	return output
}

// Sends a chat message
func (client *Client) Chat(msg string) (err error) {
	return client.SendPacket(0x03, msg)
}
