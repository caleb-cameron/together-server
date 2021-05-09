package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/abeardevil/together-engine/pb"
)

// Maps username to the user's connection.
type Connections struct {
	conns     map[string]pb.GameService_ConnectServer
	doneChans map[string]chan bool
	mutex     sync.RWMutex
}

var Conns *Connections

func init() {
	Conns = NewConnections()
}

func NewConnections() *Connections {
	return &Connections{
		conns:     map[string]pb.GameService_ConnectServer{},
		doneChans: map[string]chan bool{},
		mutex:     sync.RWMutex{},
	}
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

	if _, ok := c.doneChans[username]; ok {
		c.doneChans[username] <- true
		delete(c.doneChans, username)
	}

	return true
}

func (c *Connections) Add(username string, conn pb.GameService_ConnectServer, doneChan chan bool) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.conns[username]; ok {
		return fmt.Errorf("already have connection for %s", username)
	}

	c.conns[username] = conn

	if _, ok := c.doneChans[username]; ok {
		c.doneChans[username] <- true
		delete(c.doneChans, username)
	}

	c.doneChans[username] = doneChan

	return nil
}

func (c *Connections) Broadcast(event *pb.GameState) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for username, conn := range c.conns {
		err := conn.Send(event)
		if err != nil {
			log.Printf("Error sending update to %s: %v", username, err)
			defer c.Remove(username)
		}
	}
}
