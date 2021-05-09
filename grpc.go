package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	engine "github.com/abeardevil/together-engine"
	"github.com/abeardevil/together-engine/pb"
	"github.com/faiface/pixel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type togetherServer struct {
	pb.UnimplementedGameServiceServer
}

var GRPCServer togetherServer

func (s togetherServer) Connect(req *pb.ConnectRequest, conn pb.GameService_ConnectServer) error {
	log.Printf("Player connecting: %s.\n", req.Username)

	err := engine.PlayerList.AddPlayer(
		req.Username,
		engine.NewPlayer(req.Username,
			pixel.Vec{},
			engine.PlayerSpeed,
			engine.PlayerAcceleration,
			engine.DefaultCharacterSprite,
		),
	)

	if err != nil {
		log.Println(err)
		return err
	}

	log.Printf("New player list: %+v\n", engine.PlayerList.GetPlayers())

	doneChan := make(chan bool)

	err = Conns.Add(req.Username, conn, doneChan)

	if err != nil {
		return err
	}

	for {
		select {
		case <-doneChan:
			return nil
		}
	}
}

func (s togetherServer) SendPlayerUpdates(stream pb.GameService_SendPlayerUpdatesServer) error {
	for {
		update, err := stream.Recv()

		if err == io.EOF {
			return nil
		} else if err != nil {
			log.Printf("Error receiving player update: %v\n", err)
			return err
		}

		// log.Printf("Received update from player %s\n", update.Username)

		err = engine.PlayerList.UpdatePlayer(update.Username, engine.PlayerFromProto(update))

		if err != nil {
			log.Printf("Error applying player update: %v\n", err)
			return err
		}
	}
}

func broadcastGameState() {
	gs := buildGameState()

	if len(gs.Players) == 0 {
		return
	}

	Conns.Broadcast(gs)
}

func buildPlayerEvent(username string, player *engine.Player, eventType pb.PlayerEvent_EventType) *pb.PlayerEvent {
	e := pb.PlayerEvent{}

	e.Username = username
	e.Type = eventType
	if player != nil {
		e.Position = &pb.PlayerPosition{}

		playerPos := player.GetPosition()
		playerVel := player.GetVelocity()

		e.Position.Position = &pb.Vector{}
		e.Position.Position.X = float32(playerPos.X)
		e.Position.Position.Y = float32(playerPos.Y)

		e.Position.Velocity = &pb.Vector{}
		e.Position.Velocity.X = float32(playerVel.X)
		e.Position.Velocity.Y = float32(playerVel.Y)
	}

	return &e
}

func buildGameState() *pb.GameState {
	state := &pb.GameState{}

	players := engine.PlayerList.GetPlayers()

	connects, disconnects, updates := engine.PlayerList.GetRecents()

	for _, c := range *connects {
		e := buildPlayerEvent(c, players[c], pb.PlayerEvent_CONNECT)
		state.Players = append(state.Players, e)
	}

	for _, d := range *disconnects {
		e := buildPlayerEvent(d, players[d], pb.PlayerEvent_DISCONNECT)
		state.Players = append(state.Players, e)
	}

	for _, u := range *updates {
		if stringInSlice(u, connects) || stringInSlice(u, disconnects) {
			/*
				This player just connected or disconnected, so we've already covered
				them in the previous loops.
			*/

			continue
		}

		e := buildPlayerEvent(u, players[u], pb.PlayerEvent_UPDATE)
		state.Players = append(state.Players, e)
	}

	// log.Printf("Players in update: %+v\n", state.Players)
	// log.Printf("Connects: %+v\n", *connects)
	// log.Printf("Disconnects: %v\n", *disconnects)
	// log.Printf("Updates: %v\n", *updates)

	return state
}

func startServer() {
	log.Println("Starting GRPC server...")

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", config.ListenAddress, config.ListenPort))

	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	opts := configureGrpcOpts()
	grpcServer := grpc.NewServer(opts...)

	pb.RegisterGameServiceServer(grpcServer, GRPCServer)

	grpcServer.Serve(lis)
}

func configureGrpcOpts() []grpc.ServerOption {
	opts := []grpc.ServerOption{}

	kaep := keepalive.EnforcementPolicy{
		MinTime:             time.Second / 2.0, // Minimum time between pings
		PermitWithoutStream: true,              // Permit pings even when there is no active stream
	}
	opts = append(opts, grpc.KeepaliveEnforcementPolicy(kaep))

	// kasp := keepalive.ServerParameters{
	// 	MaxConnectionIdle:     15 * time.Second, // If a client is idle for 15 seconds, send a GOAWAY
	// 	MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before forcibly closing connections.
	// 	Time:                  5 * time.Second,  // Ping the client if it is still idle for 5 seconds to ensure the connection is still active.
	// 	Timeout:               1 * time.Second,  // Wait 1 second for the ping ack before assuming the connection is dead.
	// }
	// opts = append(opts, grpc.KeepaliveParams(kasp))

	return opts
}
