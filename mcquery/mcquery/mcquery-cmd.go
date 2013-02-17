package main

import (
	"fmt"
	"github.com/kierdavis/mc/mcquery"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <host[:port]>\n", os.Args[0])
		os.Exit(2)
	}

	addr := os.Args[1]

	if strings.Index(addr, ":") < 0 {
		addr += ":25565"
	}

	st, err := mcquery.FullStat(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("MOTD: %s\n", st.MOTD)
	fmt.Printf("Game Type: %s\n", st.GameType)
	fmt.Printf("Game ID: %s\n", st.GameID)
	fmt.Printf("Version: %s\n", st.Version)
	fmt.Printf("Server Mod: %s\n", st.ServerMod)
	fmt.Printf("Map: %s\n", st.Map)
	fmt.Printf("Players: %d/%d\n", st.NumPlayers, st.MaxPlayers)
	fmt.Printf("IP: %s\n", st.HostName)
	fmt.Printf("Port: %d\n", st.HostPort)
	fmt.Printf("\nPlugins:\n")

	for _, plugin := range st.Plugins {
		fmt.Printf("  %s\n", plugin)
	}

	fmt.Printf("\nPlayers:\n")

	for _, player := range st.Players {
		fmt.Printf("  %s\n", player)
	}
}
