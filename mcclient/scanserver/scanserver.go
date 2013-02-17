package main

import (
	"code.google.com/p/go.crypto/ssh/terminal"
	"fmt"
	"github.com/kierdavis/mc/mcclient"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Error: Not enough arguments\n")
		os.Exit(2)
	}

	addr := os.Args[1]
	result, err := mcclient.ScanServer(addr)

	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}

	motd := result.MOTD

	if len(motd) > 20 {
		motd = motd[:20]
	}

	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		motd = mcclient.ANSIEscapes(motd)
	} else {
		motd = mcclient.NoEscapes(motd)
	}

	fmt.Printf("(%d/%d) %s\n", result.PlayersOnline, result.PlayersMax, motd)
}
