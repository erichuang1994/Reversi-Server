package game

import "net"

type User struct {
	Username     string
	Addr         *net.UDPAddr
	LastModified int64
	// 所参与的游戏名
	GameName string
}

type Game struct {
	state    [8][8]int
	player   [2]*User
	stepList [][2]int
	turn     int
	leisure  bool
	Name     string
	ready    [2]bool
}

func (g *Game) Restart() {
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			g.state[i][j] = -1
		}
	}
	g.turn = 0
	g.player[0], g.player[1] = nil, nil
}
func (g *Game) Status() bool {
	return g.leisure
}

func (g *Game) Kickout(someone *User) {
	someone.GameName = ""
	for index, user := range g.player {
		if user == someone {
			g.player[index] = nil
		}
	}
}
func (g *Game) Turn() int {
	return g.turn
}

func (g *Game) Player() (*User, *User) {
	return g.player[0], g.player[1]
}

// 移动完之后返回另一位玩家的user struct
func (g *Game) Move(x int, y int) *User {
	// 把这一步记录下来
	g.stepList = append(g.stepList, [2]int{x, y})
	// 走这一步
	g.state[x][y] = g.turn
	// 逻辑处理
	// 轮到下一个人
	g.turn = (g.turn + 1) % 2
	return g.player[g.turn]
}

func (g *Game) Join(someone *User) bool {
	if g.player[0] != nil && g.player[1] != nil {
		return false
	}
	if g.player[0] == nil {
		g.player[0] = someone
	} else {
		g.player[1] = someone
	}
	someone.GameName = g.Name
	return true
}

// 两位玩家都准备好了的时候顺带返回两位user并且bool为true
func (g *Game) Ready(someone *User) (*User, *User, bool) {
	// if g.leisure == true {
	for index, user := range g.player {
		if user == someone {
			g.ready[index] = true
		}
	}
	// 两位玩家都准备好了就返回两个用户名以及true
	if g.ready[0] && g.ready[1] {
		g.leisure = false
		return g.player[0], g.player[1], true
	}
	return nil, nil, false
}
