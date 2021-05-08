package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/abeardevil/together-engine/pb"
)

// Maps username to the user's connection.
type Connections struct {
	conns map[string]pb.GameService_ConnectServer
	mutex sync.RWMutex
}

var Conns *Connections

func init() {
	Conns = &Connections{}
}

func (c *Connections) Get(username string) pb.GameService_ConnectServer {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if _, ok := c.conns[username]; !ok {
		return nil
	}

	return c.conns[username]
}

func (c *Connections) Remove(username string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.conns[username]; !ok {
		return false
	}

	delete(c.conns, username)

	return true
}

func (c *Connections) Add(username string, conn pb.GameService_ConnectServer) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.conns[username]; ok {
		return fmt.Errorf("already have connection for %s", username)
	}

	c.conns[username] = conn

	return nil
}

func (c *Connections) Broadcast(event *pb.GameState) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for username, conn := range c.conns {
		conn.Send(event)
		log.Printf("Sent game update to %s", username)
	}
}
