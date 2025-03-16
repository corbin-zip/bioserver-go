package main

import (
	"fmt"
	"sync"
	"time"
)

const (
	LOBBYPORT = 8300
	GAMEPORT  = 8690
)

func main() {
	fmt.Println("------------------------------")
	fmt.Println("-     fanmade server for     -")
	fmt.Println("- biohazard outbreak file #1 -")
	fmt.Println("-                            -")
	fmt.Println("-         corbin.zip         -")
	fmt.Println("-        go prototype        -")
	fmt.Println("------------------------------")

	// for thread-like stuff
	// go routines are like lightweight threads
	var wg sync.WaitGroup

	// set up the packethandler in its own thread
	wg.Add(1)
	packetHandler := NewPacketHandler()
	go packetHandler.Run()

	// create the lobby server thread
	lobbyServer, err := NewServerThread("192.168.1.135", LOBBYPORT, packetHandler)
	if err != nil {
		fmt.Println("Error creating lobby server:", err)
		return
	}
	wg.Add(1)
	go lobbyServer.Run(&wg)

	// create the game server thread
	gamePacketHandler := &GameServerPacketHandler{}
	gameServer := NewGameServerThread("192.168.1.135", GAMEPORT, gamePacketHandler)
	wg.Add(1)
	go gamePacketHandler.Run()
	wg.Add(1)
	go gameServer.Run(&wg)

	// allow usage
	packetHandler.SetGameServerPacketHandler(gamePacketHandler)

	// thread for the keepalivepings and cleanups
	heartbeat := NewHeartBeatThread(lobbyServer, packetHandler, gameServer, gamePacketHandler)
	go heartbeat.Run()

	time.Sleep(1 * time.Second)
	fmt.Println(time.Now().String(), "server started")

	wg.Wait()
}
