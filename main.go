// Copyright (c) 2015 Eric Huang.All Rights Reserved.

package main

import (
	"fmt"
	"net"
	. "reversi/handler"
	"strings"
	"sync"
	"time"
)

type countLock struct {
	sync.Mutex
}

func unixTime() int64 {
	return time.Now().Unix()
}

//////////////
//router    //
//          //
//////////////
func conHandler(conn *net.UDPConn, addr *net.UDPAddr, msg []byte, length int) {
	com := strings.Split(string(msg[0:length]), " ")
	fmt.Printf("receive from :%v\nmeg:%s\n", addr, msg)
	if len(com) < 2 {
		return
	}
	switch com[0] {
	case "LOGIN":
		Login(conn, addr, com[1:])
	case "OPENGAME":
		OpenGame(conn, addr, com[1:])
	case "MSG":
		Msg(conn, addr, com[1:])
	case "LIST":
		List(conn, addr, com[1:])
	case "KICKOUT":
		Kickout(conn, addr, com[1:])
	case "GAMES":
		Games(conn, addr, com[1:])
	case "WATCH":
		Watch(conn, addr, com[1:])
	case "CLOSEGAME":
		Closegame(conn, addr, com[1:])
	case "JOIN":
		Join(conn, addr, com[1:])
	case "MOVE":
		Move(conn, addr, com[1:])
	case "RESTART":
		Restart(conn, addr, com[1:])
	case "RESTARTREPLY":
		RestartReply(conn, addr, com[1:])
	case "LEAVE":
		Leave(conn, addr, com[1:])
	case "READY":
		Ready(conn, addr, com[1:])
	}
}

func main() {
	UDPAddr := net.UDPAddr{}
	UDPAddr.IP = net.IPv4(0, 0, 0, 0)
	UDPAddr.Port = 3106
	conn, err := net.ListenUDP("udp", &UDPAddr)
	if err != nil {
		fmt.Printf("ListenUDP err:%v\n", err)
	}
	for {
		msg := make([]byte, 1024)
		length, addr, err := conn.ReadFromUDP(msg)
		// time.Sleep(3 * time.Second)
		if err != nil {
			fmt.Printf("read err:%v\n", err)
		}
		go conHandler(conn, addr, msg, length)
	}
}
