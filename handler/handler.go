package handler

// Copyright (c) 2015 Eric Huang.All Rights Reserved.

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"net"
	. "reversi/game"
	"strconv"
	"strings"
	"time"
)

var (
	userList           = make(map[string]*User)
	userListByusername = make(map[string]*User)
	gameList           = make(map[string]*Game)
	rootToken          string
	restartMap         = make(map[string]chan int)
)

const admin = "root"
const password = "root"

/////////////////////////////
//Token generator          //
//current use sha1:)foreasy//
//1/12/2015                //
/////////////////////////////
func init() {
	newGame := Game{Name: "yutang"}
	newGame.Init()
	gameList["yutang"] = &newGame
}

//send  heartBeat package
func HeartBeat(conn *net.UDPConn) {
	for {
		time.Sleep(time.Second * 120)

		for token, user := range userList {
			Ping(conn, user, token)
		}
	}
}

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
	fmt.Printf("command:%v len:%d\n", msg, len(msg))
	var resp string
	if msg[0] == admin {
		fmt.Println("user root")
	}
	if len(msg) == 2 {
		if msg[0] == admin && msg[1] == password {
			rootToken = genToken(admin)
			temp := User{Username: msg[0], Addr: addr, LastModified: unixTime()}
			userList[rootToken] = &temp
			resp = "ROOT " + rootToken
		} else {
			resp = "LOGIN FAILED "
		}
		conn.WriteToUDP([]byte(resp), addr)
		return
	}
	if len(msg) == 1 {
		token := genToken(msg[0])
		_, ok := userList[token]
		resp = "LOGIN SUCCESS " + token
		if !ok {
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
		temp := []string{"GAMES"}
		for key, item := range gameList {
			temp = append(temp, key)
			if item.Status() {
				temp = append(temp, "free")
			} else {
				temp = append(temp, "busy")
			}
		}

		buffer.WriteString(strings.Join(temp, " "))
		fmt.Printf("GAMES %v", strings.Join(temp, " "))
		conn.WriteToUDP(buffer.Bytes(), addr)
	}
}

// 输出格式
// LIST username free(room:gameName)
func List(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	if _, ok := userList[cmd[0]]; ok {
		var buffer bytes.Buffer
		buffer.WriteString("LIST ")
		for _, user := range userList {
			buffer.WriteString(user.Username)
			buffer.WriteString(" ")
			if user.GameName == "" {
				buffer.WriteString("free")
			} else {
				buffer.WriteString("room:" + user.GameName)
			}
			buffer.WriteString(" ")
		}
		conn.WriteToUDP(buffer.Bytes(), addr)
	}
}

func Join(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	user, ok := userList[cmd[1]]
	if ok {
		game, _ := gameList[cmd[0]]
		if game.Join(user) {
			conn.WriteToUDP([]byte("JOIN Success"), addr)
			return
		}
	}
	conn.WriteToUDP([]byte("JOIN Fail"), addr)

}

// 当两个玩家都准备的时候发送START让游戏开始
func Ready(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	user, ok := userList[cmd[0]]
	if ok {
		user1, user2, ok := gameList[cmd[0]].Ready(user)
		conn.WriteToUDP([]byte("READY SUCCESS"), userListByusername[user1.Username].Addr)
		if ok {
			conn.WriteToUDP([]byte("START black"), userListByusername[user1.Username].Addr)
			conn.WriteToUDP([]byte("START white"), userListByusername[user2.Username].Addr)
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
			conn.WriteToUDP([]byte("OPENGAME FAIL"), addr)
		} else {
			newGame := Game{Name: cmd[0]}
			newGame.Init()
			gameList[cmd[0]] = &newGame
			conn.WriteToUDP([]byte("OPENGAME SUCCESS"), addr)
		}
	}
}

func Watch(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	if cmd[1] == rootToken {
		game, _ := gameList[cmd[0]]
		root, _ := userList[cmd[1]]
		game.SetWatcher(root)
		steps := game.Steps()
		var buffer bytes.Buffer
		for i := 0; i < len(steps); i++ {
			buffer.WriteString(fmt.Sprintf("(%d,%d)", steps[i][0], steps[i][1]))
		}
		conn.WriteToUDP(buffer.Bytes(), root.Addr)
		root.GameName = cmd[0]
	}
}

func Leave(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	user, ok := userList[cmd[1]]
	if ok {
		game, _ := gameList[user.GameName]
		another, ok := game.Leave(user)
		if ok {
			conn.WriteToUDP([]byte("LEAVE"), another.Addr)
		}
	}
}

func CloseGame(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	if cmd[1] == rootToken {
		game, _ := gameList[cmd[0]]
		user1, user2 := game.Player()
		game.Close()
		delete(gameList, cmd[0])
		if user1 != nil {
			conn.WriteToUDP([]byte("CLOSE"), user1.Addr)
		}
		if user2 != nil {
			conn.WriteToUDP([]byte("CLOSE"), user2.Addr)
		}
		conn.WriteToUDP([]byte("CLOSE "+cmd[0]+" SUCCESS"), addr)
	}
}

////////////////////////////////////////
// function Ping                      //
//client should return pong           //
//then server update user.lastModified//
////////////////////////////////////////
func Ping(conn *net.UDPConn, user *User, token string) bool {
	conn.WriteToUDP([]byte("PING"), user.Addr)
	time.Sleep(time.Second)
	if unixTime()-user.LastModified > 2 {
		// 清除不在线用户
		game, ok := gameList[user.GameName]
		if ok {
			game.Kickout(user)
		}
		delete(userList, token)
		return false
	}
	return true
}

func Pong(conn *net.UDPConn, addr *net.UDPAddr, cmd []string) {
	user, ok := userList[cmd[0]]
	if ok {
		user.LastModified = unixTime()
	}
}
