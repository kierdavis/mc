package mcquery

import (
	"bytes"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

const MaxRetries = 3
const Timeout = time.Second * 5

type Connection struct {
	Conn      net.Conn
	ID        uint32
	Retries   int
	Challenge uint32
}

type Stat struct {
	MOTD       string
	GameType   string
	GameID     string
	Version    string
	ServerMod  string
	Map        string
	NumPlayers int
	MaxPlayers int
	HostPort   int
	HostName   string
	Plugins    []string
	Players    []string
}

func getUint32(b []byte) (n uint32) {
	n = uint32(b[0]) << 24
	n |= uint32(b[1]) << 16
	n |= uint32(b[2]) << 8
	n |= uint32(b[3])
	return n
}

func putUint32(b []byte, n uint32) {
	b[0] = byte(n >> 24)
	b[1] = byte(n >> 16)
	b[2] = byte(n >> 8)
	b[3] = byte(n)
}

func Connect(addr string) (c *Connection, err error) {
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return nil, err
	}

	c = &Connection{
		Conn:      conn,
		ID:        0,
		Retries:   0,
		Challenge: 0,
	}

	err = c.Handshake()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Connection) WritePacket(t byte, payload []byte) (err error) {
	message := make([]byte, len(payload)+7)
	message[0] = 0xFE
	message[1] = 0xFD
	message[2] = t

	putUint32(message[3:7], c.ID)
	copy(message[7:], payload)

	_, err = c.Conn.Write(message)
	if err != nil {
		return err
	}

	return c.Conn.SetReadDeadline(time.Now().Add(Timeout))
}

func (c *Connection) ReadPacket() (t byte, id uint32, payload []byte, err error) {
	buffer := make([]byte, 2048)
	n, err := c.Conn.Read(buffer)
	if err != nil {
		return 0, 0, nil, err
	}

	buffer = buffer[:n]

	t = buffer[0]
	id = getUint32(buffer[1:5])
	payload = buffer[5:]

	return t, id, payload, nil
}

func (c *Connection) Handshake() (err error) {
	c.ID += 1
	err = c.WritePacket(9, nil)
	if err != nil {
		return err
	}

	_, _, payload, err := c.ReadPacket()
	if err != nil {
		e, ok := err.(net.Error)
		if ok && e.Timeout() {
			c.Retries++

			if c.Retries == MaxRetries {
				return fmt.Errorf("Retry limit reached - server down?")
			}

			return c.Handshake()
		}

		return err
	}

	challenge, err := strconv.ParseUint(string(payload[:len(payload)-1]), 10, 32)
	if err != nil {
		return err
	}

	c.Retries = 0
	c.Challenge = uint32(challenge)

	return nil
}

func (c *Connection) BasicStat() (r *Stat, err error) {
	packet := make([]byte, 4)
	putUint32(packet, c.Challenge)

	err = c.WritePacket(0, packet)
	if err != nil {
		return nil, err
	}

	_, _, payload, err := c.ReadPacket()
	if err != nil {
		err = c.Handshake()
		if err != nil {
			return nil, err
		}

		return c.BasicStat()
	}

	r = new(Stat)
	parts := bytes.SplitN(payload, []byte{0}, 6)

	r.MOTD = string(parts[0])
	r.GameType = string(parts[1])
	r.Map = string(parts[2])

	numPlayers, err := strconv.ParseInt(string(parts[3]), 10, 0)
	if err != nil {
		return nil, err
	}

	maxPlayers, err := strconv.ParseInt(string(parts[4]), 10, 0)
	if err != nil {
		return nil, err
	}

	r.NumPlayers = int(numPlayers)
	r.MaxPlayers = int(maxPlayers)

	payload = parts[5]
	r.HostPort = int(uint16(payload[0]) | (uint16(payload[1]) << 8))
	r.HostName = string(payload[2 : len(payload)-1])

	return r, nil
}

func (c *Connection) FullStat() (r *Stat, err error) {
	packet := make([]byte, 8)
	putUint32(packet, c.Challenge)

	err = c.WritePacket(0, packet)
	if err != nil {
		return nil, err
	}

	_, _, payload, err := c.ReadPacket()
	if err != nil {
		err = c.Handshake()
		if err != nil {
			return nil, err
		}

		return c.FullStat()
	}

	payload = payload[11:]
	p := bytes.Index(payload, []byte("\x00\x00\x01player_\x00\x00"))
	itemsPayload := payload[:p]
	playersPayload := payload[p+12:]

	r = new(Stat)

	itemsParts := bytes.Split(itemsPayload, []byte{0x00})
	for i := 0; i < len(itemsParts); i += 2 {
		key := string(itemsParts[i])
		value := string(itemsParts[i+1])

		switch key {
		case "hostname":
			r.MOTD = value
		case "gametype":
			r.GameType = value
		case "game_id":
			r.GameID = value
		case "version":
			r.Version = value
		case "plugins":
			r.ServerMod, r.Plugins = parsePlugins(value)
		case "map":
			r.Map = value

		case "numplayers":
			v, err := strconv.ParseInt(value, 10, 0)
			if err != nil {
				return nil, err
			}

			r.NumPlayers = int(v)

		case "maxplayers":
			v, err := strconv.ParseInt(value, 10, 0)
			if err != nil {
				return nil, err
			}

			r.MaxPlayers = int(v)

		case "hostport":
			v, err := strconv.ParseInt(value, 10, 0)
			if err != nil {
				return nil, err
			}

			r.HostPort = int(v)

		case "hostip":
			r.HostName = value
		}
	}

	if len(playersPayload) > 2 {
		playersPayload = playersPayload[:len(playersPayload)-2]
		r.Players = strings.Split(string(playersPayload), "\x00")
	}

	sort.Sort(caseInsensitiveStrings(r.Plugins))
	sort.Sort(caseInsensitiveStrings(r.Players))

	return r, nil
}

func BasicStat(addr string) (r *Stat, err error) {
	conn, err := Connect(addr)
	if err != nil {
		return nil, err
	}

	return conn.BasicStat()
}

func FullStat(addr string) (r *Stat, err error) {
	conn, err := Connect(addr)
	if err != nil {
		return nil, err
	}

	return conn.FullStat()
}

func parsePlugins(s string) (serverMod string, plugins []string) {
	p := strings.Index(s, ": ")
	if p < 0 {
		return s, nil
	}

	serverMod = s[:p]
	plugins = strings.Split(s[p+2:], "; ")
	return serverMod, plugins
}

type caseInsensitiveStrings []string

func (p caseInsensitiveStrings) Len() (length int) {
	return len(p)
}

func (p caseInsensitiveStrings) Less(i, j int) (result bool) {
	return strings.ToLower(p[i]) < strings.ToLower(p[j])
}

func (p caseInsensitiveStrings) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
