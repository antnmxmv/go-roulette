package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go-rooms/games"
	"go-rooms/lib/rooms"
	"io/ioutil"
	"log"
	"strconv"
)

/**
	handler.
	Returns JSON token and role as struct
 */
func findGame(gameName string) func(c *gin.Context) {
	return func(c *gin.Context) {
		var g struct {
			Token string
			Role  int
		}
		g.Token, g.Role = rooms.FindRoom(gameName)
		c.JSON(200, g)
		c.AbortWithStatus(200)
	}
}

/**
	hadler.
	Returns JSON field to client as 2d array
 */
func getGrid(c *gin.Context) {
	token, ok := c.Params.Get("room")
	if !ok {
		c.AbortWithStatus(400)
		return
	}
	if r, ok := rooms.GetRoom(token); ok {
		c.JSON(200, r.Game.GetGrid())
	} else {
		c.AbortWithError(200, errors.New("waiting for another player"))
	}
}

/**
	handler.
	Accepts parameters as sarray of ints.
	Returns http codes
 */
func action(c *gin.Context) {
	var params []int
	token, ok1 := c.Params.Get("room")
	role, ok2 := c.Params.Get("role")
	if !(ok1 && ok2) {
		c.AbortWithStatus(400)
		return
	}
	intRole, err := strconv.Atoi(role)
	if err != nil {
		c.AbortWithError(400, err)
	}
	c.ShouldBindJSON(&params)
	params = append([]int{intRole}, params...)
	r, ok := rooms.GetRoom(token)
	if !ok {
		c.AbortWithStatus(404)
		return
	}
	ok, err = r.TurnHandler(params...)
	if err != nil {
		c.AbortWithStatus(200)
		return
	}
	c.JSON(200, ok)
	r.FirstSocket.WriteJSON(r.Game.GetGrid())
	if r.SecondSocket != nil {
		r.SecondSocket.WriteJSON(r.Game.GetGrid())
	}
	rooms.SetRoom(token, r)
	c.AbortWithStatus(200)
	c.JSON(200, "{}")
}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

/**
	Web-socket handler
	Handle connection and stores it to specific room
 */
func wshandler(c *gin.Context) {
	conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	token, ok := c.Params.Get("room")
	if !ok {
		c.AbortWithStatus(400)
		return
	}
	rooms.NewConnection(string(token), conn)
}

/**
	handelr
	Returns string number of player by rooms token
 */
func getTurn(c *gin.Context) {
	token, ok := c.Params.Get("room")
	if !ok {
		c.AbortWithStatus(400)
		return
	}
	r, ok := rooms.GetRoom(token)
	if !ok {
		c.AbortWithStatus(404)
		return
	}
	c.Writer.Write([]byte(strconv.Itoa(r.Turn)))
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	router.GET("/ws/:room/", wshandler)

	router.GET("/api/:room/", getGrid)
	router.GET("/api/:room/next_turn/", getTurn)
	router.GET("/new/tic_tac_toe/", findGame("tic_tac_toe"))
	router.GET("/new/hexapawn/", findGame("hexapawn"))
	router.POST("/api/:room/turn/:role/", action)

	files, err := ioutil.ReadDir("./games")

	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if f.IsDir() {
			if games.GetInstance(f.Name()) == nil {
				fmt.Printf("Warning: Unmapped folder %q in /games/ directory. Please write a case in names.go\n", f.Name())
				continue
			}
			router.StaticFile("/"+f.Name()+"/", "public/"+f.Name()+"/main.html")
			router.Static("/"+f.Name()+"/assets/", "public/"+f.Name())
		}
	}

	router.Run(":8080")
}