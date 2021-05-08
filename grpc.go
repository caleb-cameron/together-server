package main

import (
	"log"
	"net"

	engine "github.com/abeardevil/together-engine"
	"github.com/abeardevil/together-engine/pb"
	"google.golang.org/grpc"
)

type togetherServer struct {
	pb.UnimplementedGameServiceServer
}

var GRPCServer togetherServer

// Maps username to the user's connection.
var Connections map[string]*pb.GameService_ConnectServer

func (s togetherServer) Connect(req *pb.ConnectRequest, stream pb.GameService_ConnectServer) error {
	log.Printf("Got ConnectRequest from %s.\n", req.Username)

	err := PlayerList.AddPlayer(req.Username)

	if err != nil {
		return err
	}

	stream.Send(buildGameState())

	return nil
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

	players := PlayerList.GetPlayers()

	connects, disconnects := PlayerList.GetRecents()

	for _, c := range *connects {
		e := buildPlayerEvent(c, players[c], pb.PlayerEvent_CONNECT)
		state.Players = append(state.Players, e)
	}

	for _, d := range *disconnects {
		e := buildPlayerEvent(d, players[d], pb.PlayerEvent_DISCONNECT)
		state.Players = append(state.Players, e)
	}

	for username, player := range players {
		if stringInSlice(username, connects) || stringInSlice(username, disconnects) {
			/*
				This player just connected or disconnected, so we've already covered
				them in the previous loops.
			*/

			break
		}

		e := buildPlayerEvent(username, player, pb.PlayerEvent_UPDATE)
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
