package main

import (
	"log"
	"net"

	engine "github.com/abeardevil/together-engine"
	"github.com/abeardevil/together-engine/pb"
	"github.com/faiface/pixel"
	"google.golang.org/grpc"
)

type togetherServer struct {
	pb.UnimplementedGameServiceServer
}

var GRPCServer togetherServer

func (s togetherServer) Connect(req *pb.ConnectRequest, conn pb.GameService_ConnectServer) error {
	log.Printf("Got ConnectRequest from %s.\n", req.Username)

	err := engine.PlayerList.AddPlayer(req.Username, engine.NewPlayer(pixel.Vec{}, engine.PlayerSpeed, engine.PlayerAcceleration, engine.DefaultCharacterSprite))

	if err != nil {
		return err
	}

	Conns.Add(req.Username, conn)

	broadcastGameState()

	return nil
}

func broadcastGameState() {
	Conns.Broadcast(buildGameState())
}

func buildPlayerEvent(username string, player *engine.Player, eventType pb.PlayerEvent_EventType) *pb.PlayerEvent {
	e := pb.PlayerEvent{}
	e.Type = pb.PlayerEvent_CONNECT
	e.Position = &pb.PlayerPosition{}
	playerPos := player.GetPosition()
	playerVel := player.GetVelocity()

	e.Position.Position = &pb.Vector{}
	e.Position.Position.X = float32(playerPos.X)
	e.Position.Position.Y = float32(playerPos.Y)

	e.Position.Velocity = &pb.Vector{}
	e.Position.Velocity.X = float32(playerVel.X)
	e.Position.Velocity.Y = float32(playerVel.Y)

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

			break
		}

		e := buildPlayerEvent(u, players[u], pb.PlayerEvent_UPDATE)
		state.Players = append(state.Players, e)
	}

	return state
}

func startServer() {
	log.Println("Starting GRPC server...")

	lis, err := net.Listen("tcp", "localhost:9000")

	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)

	pb.RegisterGameServiceServer(grpcServer, GRPCServer)

	grpcServer.Serve(lis)
}
