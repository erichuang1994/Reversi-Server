package handler

// Copyright (c) 2015 Eric Huang.All Rights Reserved.

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"net"
	. "reversi/game"
	"strconv"
	"time"
)

var (
	userList           map[string]*User
	userListByusername map[string]*User
	gameList           map[string]*Game
	rootToken          string
	playList           map[string]*Game
	restartMap         map[string]chan int
)

const admin = "root"
const password = "root"

/////////////////////////////
//Token generator          //
//current use sha1:)foreasy//
//1/12/2015                //
/////////////////////////////
func genToken(username string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(username)))
}

func unixTime() int64 {
	return time.Now().Unix()
}

func Msg(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	if len(cmd) == 3 && cmd[2] == rootToken {
		user, _ := userList[cmd[0]]

		conn.WriteToUDP([]byte(cmd[1]), user.Addr)
	} else if len(cmd) == 2 && cmd[1] == rootToken {
		var buffer bytes.Buffer
		buffer.WriteString("MSG " + cmd[0])
		for _, user := range userList {
			conn.WriteToUDP(buffer.Bytes(), user.Addr)
		}
	}
}

func Kickout(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	if cmd[1] == rootToken {
		user := userList[cmd[0]]
		gameList[user.GameName].Kickout(user)
	}
}
func Login(conn *net.UDPConn, addr *net.UDPAddr, msg []string) {
	fmt.Printf("com:%v\n", msg)
	if msg[0] == admin {
		fmt.Println("user root")
	}
	if len(msg) == 2 && msg[0] == admin && msg[1] == password {
		rootToken = genToken(admin)
		conn.WriteToUDP([]byte(rootToken), addr)
		return
	}
	if len(msg) == 1 {
		token := genToken(msg[0])
		_, ok := userList[token]
		var resp string
		if ok {
			// 用户已经登录或者重复登录
			resp = token
		} else {
			temp := User{Username: msg[0], Addr: addr, LastModified: unixTime()}
			userList[token] = &temp
			userListByusername[msg[0]] = &temp
		}
		conn.WriteToUDP([]byte(resp), addr)
		return
	}
}
func Games(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	if _, ok := userList[cmd[0]]; ok {
		var buffer bytes.Buffer
		for key := range playList {
			buffer.WriteString(key)
			buffer.WriteString(" ")
		}
		conn.WriteToUDP(buffer.Bytes(), addr)
	}
}

// 输出格式
// username free(room:gameName) \n
func List(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	if _, ok := userList[cmd[0]]; ok {
		var buffer bytes.Buffer
		for _, user := range userList {
			buffer.WriteString(user.Username)
			buffer.WriteString(" ")
			if user.GameName == "" {
				buffer.WriteString("free")
			} else {
				buffer.WriteString("room:" + user.GameName)
			}
			buffer.WriteString("\n")
		}
		conn.WriteToUDP(buffer.Bytes(), addr)
	}
}

func Join(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	user, ok := userList[cmd[1]]
	if ok {
		game, _ := gameList[user.GameName]
		if game.Join(user) {
			conn.WriteToUDP([]byte("Success"), addr)
		}
	} else {
		conn.WriteToUDP([]byte("Fail"), addr)
	}
}

// 当两个玩家都准备的时候发送START让游戏开始
func Ready(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	user, ok := userList[cmd[1]]
	if ok {
		user1, user2, ok := gameList[cmd[0]].Ready(user)
		if ok {
			conn.WriteToUDP([]byte("START"), userListByusername[user1.Username].Addr)
			conn.WriteToUDP([]byte("START"), userListByusername[user2.Username].Addr)
		}
	}
}

// Move x y移动完应该通知另一位玩家
func Move(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	_, ok1 := userList[cmd[3]]
	game, ok2 := gameList[cmd[0]]
	if ok1 && ok2 {
		x, _ := strconv.Atoi(cmd[1])
		y, _ := strconv.Atoi(cmd[2])
		user := game.Move(x, y)
		conn.WriteToUDP([]byte(fmt.Sprintf("MOVE %d %d", x, y)), user.Addr)
	}
}

func Restart(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	user, ok := userList[cmd[0]]
	if ok {
		user1, user2 := gameList[user.GameName].Player()
		var another *User
		if user1 == user {
			another = user2
		} else {
			another = user1
		}
		restartMap[user.GameName] = make(chan int)
		// 询问另一个玩家
		conn.WriteToUDP([]byte("RESTART"), another.Addr)
		ok := <-restartMap[user.GameName]
		if ok == 1 {
			conn.WriteToUDP([]byte("OK"), addr)
		} else {
			conn.WriteToUDP([]byte("NO"), addr)
		}
		delete(restartMap, user.GameName)
	}
}

func RestartReply(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	user, ok := userList[cmd[1]]
	if ok {
		flag, ok := restartMap[user.GameName]
		if ok {
			ans, _ := strconv.Atoi(cmd[0])
			flag <- ans
		}
	}
}

// 房间已经存在的时候返回403 成功返回200
func OpenGame(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	if cmd[1] == rootToken {
		if _, ok := gameList[cmd[0]]; ok {
			// 该名字命名的游戏已经存在
			conn.WriteToUDP([]byte("403"), addr)
		} else {
			newGame := Game{Name: cmd[0]}
			gameList[cmd[0]] = &newGame
			conn.WriteToUDP([]byte("200"), addr)
		}
	}
}

////////////////////////////////////////
// function Ping                      //
//client should return pong           //
//then server update user.lastModified//
////////////////////////////////////////
func Ping(conn *net.UDPConn, user User) bool {
	conn.WriteToUDP([]byte("PING"), user.Addr)
	time.Sleep(time.Second)
	if unixTime()-user.LastModified > 2 {
		return false
	}
	return true
}
