package game

import (
	"fmt"
	"net"
)

type User struct {
	Username     string
	Addr         *net.UDPAddr
	LastModified int64
	// 所参与的游戏名
	GameName string
}

type Game struct {
	board     [8][8]int // -1为空 0 为黑子 1 为白子
	player    [2]*User  //user0为黑方 user1为白方
	stepList  [][2]int
	turn      int
	leisure   bool
	Name      string
	ready     [2]bool
	watcher   *User
	direction [8][2]int
	black     int
	white     int
}

func (g *Game) Init() {
	g.black = 0
	g.white = 1
	g.leisure = true
	g.player[0], g.player[1] = nil, nil
	g.watcher = nil
	g.direction = [8][2]int{{1, 0}, {1, 1}, {1, -1}, {0, 1}, {0, -1}, {-1, 1}, {-1, 0}, {-1, -1}}
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			g.board[i][j] = -1
		}
	}
	g.board[3][3], g.board[4][4] = g.black, g.black
	g.board[3][4], g.board[4][3] = g.white, g.white
	g.watcher = nil
}

func (g *Game) Restart() {
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			g.board[i][j] = -1
		}
	}
	g.board[3][3], g.board[4][4] = g.black, g.black
	g.board[3][4], g.board[4][3] = g.white, g.white
	g.turn = 0
	var temp [][2]int
	g.stepList = temp
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
	if g.leisure == false {
		g.leisure = true
	}
}
func (g *Game) Turn() *User {
	return g.player[g.turn]
}

func (g *Game) Player() (*User, *User) {
	return g.player[0], g.player[1]
}

// 移动完之后返回两位玩家的*User
// 通常情况下最后一个参数为false当为true时表明游戏结束，此时第一个参数为赢家
// 第二个参数的败者
// 第三个返回值通常也为nil,当存在观战的人的时候返回观战的人的*User
func (g *Game) Move(x int, y int) (*User, *User, *User, bool) {
	// 把这一步记录下来
	g.stepList = append(g.stepList, [2]int{x, y})
	// 走这一步
	turn := g.turn
	g.board[x][y] = turn
	for i := 0; i < len(g.direction); i++ {
		for loc := g.add([2]int{x, y}, g.direction[i]); 0 <= loc[0] && loc[0] < 8 && 0 < loc[1] && loc[1] < 8; loc = g.add(loc, g.direction[i]) {
			if g.getPoint(loc) == (turn+1)%2 {
				continue
			} else if g.getPoint(loc) == turn {
				for temp := g.add([2]int{x, y}, g.direction[i]); !g.equalPoint(temp, loc); temp = g.add(temp, g.direction[i]) {
					g.board[temp[0]][temp[1]] = turn
				}
			}
		}
	}
	// g.board[x][y] = g.turn
	// 逻辑处理
	if g.movaable((turn+1)%2) == true {
		g.turn = (g.turn + 1) % 2
	} else if g.movaable(turn) == false {
		// 两个人都不能走的时候游戏结束，结算一下谁赢了
		white := 0
		black := 0
		for x := 0; x < 8; x++ {
			for y := 0; y < 8; y++ {
				if g.board[x][y] == 0 {
					black++
				} else if g.board[x][y] == 1 {
					white++
				}
			}
		}
		if white > black {
			return g.player[1], g.player[0], g.watcher, true
		}
		return g.player[0], g.player[1], g.watcher, true
	}
	return g.player[0], g.player[1], g.watcher, false
}

// 判断某个人能不能走
func (g *Game) movaable(turn int) bool {
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			if g.testMove(x, y, turn) == true {
				return true
			}
		}
	}
	return false
}

func (g *Game) testMove(x int, y int, turn int) bool {
	if !(0 <= x && x < 8 && 0 <= y && y < 8) || g.board[x][y] != -1 {
		return false
	}
	fmt.Printf("test move %d %d\n", x, y)
	for i := 0; i < len(g.direction); i++ {
		for loc := g.add([2]int{x, y}, g.direction[i]); 0 <= loc[0] && loc[0] < 8 && 0 <= loc[1] && loc[1] < 8; loc = g.add(loc, g.direction[i]) {
			// fmt.Printf("(%d,%d) %d\n", loc[0], loc[1], g.getPoint(loc))
			if g.getPoint(loc) == (turn+1)%2 {
				continue
			} else if g.getPoint(loc) == turn {
				for temp := g.add([2]int{x, y}, g.direction[i]); !g.equalPoint(temp, loc); temp = g.add(temp, g.direction[i]) {
					// fmt.Printf("(%d,%d)  SUCCESS\n", temp[0], temp[1])
					return true
				}
			}
			// 是空白时退出这一轮循环
			break
		}
	}
	return false
}
func (g *Game) Join(someone *User) bool {
	if g.leisure == false {
		// 人满不能加了
		return false
	}
	for index, user := range g.player {
		if user == someone {
			return false
		}
		if user == nil {
			g.player[index] = someone
			break
		}
	}
	// if g.player[0] == nil {
	// 	g.player[0] = someone
	// } else {
	// 	g.player[1] = someone
	// }

	if g.player[0] != nil && g.player[1] != nil {
		g.leisure = false
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
	// 两位玩家都准备好了就返回两个用户名以及true,表明游戏开始
	if g.ready[0] && g.ready[1] {
		g.leisure = false
		return g.player[0], g.player[1], true
	}
	return nil, nil, false
}

// 将user的GameName清掉
func (g *Game) Close() {
	for index, user := range g.player {
		if user != nil {
			user.GameName = ""
			g.player[index] = nil
		}
	}
}

func (g *Game) Steps() [][2]int {
	return g.stepList
}

func (g *Game) SetWatcher(user *User) {
	g.watcher = user
}

func (g *Game) Watch() (*User, bool) {
	if g.watcher != nil {
		return g.watcher, true
	}
	return nil, false
}

// 如果是玩家退出返回另一个玩家与true
func (g *Game) Leave(someone *User) (*User, bool) {
	if g.watcher == someone {
		g.watcher = nil
		return nil, false
	}
	for index, user := range g.player {
		if user == someone {
			g.player[index] = nil
			return g.player[(index+1)%2], true
		}
	}
	return nil, false
}

func (g *Game) add(loc [2]int, dir [2]int) [2]int {
	loc[0], loc[1] = loc[0]+dir[0], loc[1]+dir[1]
	return loc
}

func (g *Game) getPoint(loc [2]int) int {
	return g.board[loc[0]][loc[1]]
}

func (g *Game) equalPoint(loc [2]int, x [2]int) bool {
	return loc[0] == x[0] && loc[1] == x[1]
}
func (g *Game) showGame() {
	fmt.Printf("%v\n", g.board)
}

func init() {
	test := Game{Name: "fuck"}
	test.Init()
	test.showGame()
	if test.testMove(2, 2, 0) == true {
		fmt.Println("so \n")
	}
}
