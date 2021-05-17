package main

import (
	"context"
	"errors"
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
var TileUpdateQueue *engine.ConcurrentQueue

func init() {
	TileUpdateQueue = engine.NewConcurrentQueue()
}

func (s togetherServer) Register(ctx context.Context, req *pb.UserRegistration) (*pb.LoginResponse, error) {
	if req.Username == "" || req.Password == "" || req.Email == "" {
		return nil, errors.New("Username, password and email required")
	}

	user, err := NewUserAccount(req.Username, req.Password, req.Email)

	if err != nil {
		if err == ErrUsernameTaken {
			log.Printf("User registration failed because username %s is taken.", req.Username)
		}

		log.Printf("User registration failed: %v", err)
		return nil, err
	}

	if user == nil {
		log.Printf("User registration produced nil user: %v", err)
		return nil, errors.New("User registration produced nil user.")
	}

	token, err := user.GenerateToken()
	resp := pb.LoginResponse{
		Username: req.Username,
		Token:    token,
		Success:  true,
	}

	if err != nil || token == "" {
		err = fmt.Errorf("User registration succeeded but failed to create a JWT token: %v", err)
		resp.Error = "Login succeeded but failed to create a token -- please report this error."
		return &resp, err
	}

	return &resp, nil
}

func (s togetherServer) Login(ctx context.Context, req *pb.UserLogin) (*pb.LoginResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, errors.New("Username and password required.")
	}

	user, err := Login(req.Username, req.Password)

	if err != nil {
		if err == ErrUserDoesNotExist {
			log.Printf("User login failed because user %s does not exist.", req.Username)
		}

		log.Printf("User login failed: %v", err)
		return nil, err
	}

	resp := pb.LoginResponse{
		Username: req.Username,
	}

	if user == nil {
		resp.Success = false
		resp.Error = "Invalid username or password"
		return &resp, nil
	}

	token, err := user.GenerateToken()

	if err != nil || token == "" {
		err = fmt.Errorf("User login succeeded but failed to create a JWT token: %v", err)
		log.Println(err)
		resp.Error = err.Error()
		return &resp, err
	}

	resp.Token = token

	return &resp, nil
}

func (s togetherServer) Connect(req *pb.ConnectRequest, conn pb.GameService_ConnectServer) error {
	if req.Token == "" {
		return errors.New("No auth token provided in connect request")
	}

	user, err := GetUserByToken(req.Token)

	if err != nil || user == nil {
		log.Printf("User connect failed: bad token: %v", err)
		return err
	}

	err = engine.PlayerList.AddPlayer(
		user.Username,
		engine.NewPlayer(user.Username,
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

	err = Conns.Add(user.Username, conn, doneChan)

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

		err = engine.PlayerList.UpdatePlayer(update.Username, engine.PlayerFromProto(update))

		if err != nil {
			log.Printf("Error applying player update: %v\n", err)
			return err
		}
	}
}

func broadcastGameState() {
	gs := buildGameState()

	if len(gs.Players) == 0 && len(gs.TileUpdates) == 0 {
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

			// continue
		}

		e := buildPlayerEvent(u, players[u], pb.PlayerEvent_UPDATE)
		state.Players = append(state.Players, e)
	}

	state.TileUpdates = []*pb.TileUpdate{}

	for tu := range TileUpdateQueue.Iter() {
		state.TileUpdates = append(state.TileUpdates, tu.Value.(*pb.TileUpdate))
	}

	return state
}

func (s togetherServer) LoadChunk(ctx context.Context, vec *pb.Vector) (*pb.Chunk, error) {
	chunkX := int(vec.X)
	chunkY := int(vec.Y)

	log.Printf("Got a LoadChunk request for chunk (%d,%d)\n", chunkX, chunkY)

	chunk := engine.GWorld.GetChunk(chunkX, chunkY)

	if chunk == nil {
		engine.GWorld.LoadOrCreateChunk(chunkX, chunkY)
		chunk = engine.GWorld.GetChunk(chunkX, chunkY)
	}

	b, err := chunk.Encode()

	if err != nil {
		log.Printf("Failed to load chunk (%d,%d): %v", chunkX, chunkY, err)
		return nil, err
	}

	resp := pb.Chunk{}
	resp.Coordinates = &pb.Vector{X: float32(chunkX), Y: float32(chunkY)}
	resp.ChunkData = b

	return &resp, nil
}

func (s togetherServer) UpdateTile(ctx context.Context, tu *pb.TileUpdate) (*pb.Ack, error) {
	chunkX := int(tu.ChunkCoordinates.X)
	chunkY := int(tu.ChunkCoordinates.Y)
	tileX := int(tu.TileCoordinates.X)
	tileY := int(tu.TileCoordinates.Y)

	log.Printf("Got tile update: %+v", tu)

	c := engine.GWorld.GetChunk(chunkX, chunkY)

	if c == nil {
		return nil, errors.New(fmt.Sprintf("Tried to update nil chunk: %d, %d", chunkX, chunkY))
	}

	tile := engine.DeserializeTile(tu.TileData)
	tile.Chunk = c

	log.Printf("Deserialized tile: %+v", tile)

	c.ReplaceTile(tileX, tileY, tile)
	c.PersistToDisk()

	TileUpdateQueue.Push(tu)

	return &pb.Ack{}, nil
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
