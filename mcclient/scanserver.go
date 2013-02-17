package mcclient

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode/utf8"
)

type ScanServerResult struct {
	ProtocolVersion  int
	MinecraftVersion string
	MOTD             string
	PlayersOnline    int
	PlayersMax       int
}

func ScanServer(addr string) (result *ScanServerResult, err error) {
	if strings.Index(addr, ":") < 0 {
		addr += ":25565"
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, err = conn.Write([]byte{0xFE})
	if err != nil {
		return nil, err
	}

	data := make([]byte, 1024)
	n, err := conn.Read(data)
	if err != nil {
		return nil, err
	}

	if data[0] != 0xFF {
		return nil, fmt.Errorf("Expected kick packet (0xFF)")
	}

	realLen := 0
	runes := make([]rune, 0, (n-3)/2)

	for i := 3; i < n; i += 2 {
		r := (rune(data[i]) << 8) | rune(data[i+1])
		runes = append(runes, r)
		realLen += utf8.RuneLen(r)
	}

	b := make([]byte, realLen)
	pos := 0

	for _, r := range runes {
		pos += utf8.EncodeRune(b[pos:], r)
	}

	s := string(b)

	/*
		if strings.HasPrefix(s, "\xc2\xa7") {
			parts := strings.Split(s, "\x00")

			protocolVersion, err := strconv.ParseInt(parts[1], 10, 0)
			if err != nil {
				return nil, err
			}

			playersOnline, err := strconv.ParseInt(parts[4], 10, 0)
			if err != nil {
				return nil, err
			}

			playersMax, err := strconv.ParseInt(parts[5], 10, 0)
			if err != nil {
				return nil, err
			}

			result = &ScanServerResult{
				ProtocolVersion:  int(protocolVersion),
				MinecraftVersion: parts[2],
				MOTD:             parts[3],
				PlayersOnline:    int(playersOnline),
				PlayersMax:       int(playersMax),
			}

		} else {*/

	p := strings.LastIndex(s, "\xc2\xa7")
	if p < 0 {
		return nil, fmt.Errorf("Bad response message format")
	}

	playersMaxStr := s[p+2:]
	s = s[:p]

	p = strings.LastIndex(s, "\xc2\xa7")
	if p < 0 {
		return nil, fmt.Errorf("Bad response message format")
	}

	playersOnlineStr := s[p+2:]
	s = s[:p]

	playersOnline, err := strconv.ParseInt(playersOnlineStr, 10, 0)
	if err != nil {
		return nil, err
	}

	playersMax, err := strconv.ParseInt(playersMaxStr, 10, 0)
	if err != nil {
		return nil, err
	}

	result = &ScanServerResult{
		ProtocolVersion:  39,
		MinecraftVersion: "Unknown version",
		MOTD:             strings.Replace(s, "\xc3\x82\xc2\xa7", "\xc2\xa7", -1),
		PlayersOnline:    int(playersOnline),
		PlayersMax:       int(playersMax),
	}

	//}

	return result, nil
}
