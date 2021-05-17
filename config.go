package main

import (
	"fmt"

	engine "github.com/abeardevil/together-engine"
	"github.com/spf13/viper"
)

type Config struct {
	WorldSeed string

	ListenAddress string
	ListenPort    string

	DataDir   string
	ChunksDir string
	BadgerDir string
	ChunkSize int

	WorldMaxAltitude int
	ChunkLoadRadius  float64
	ChunkLoadPadding float64

	AuthTokenKey string
}

var config Config

func initConfigs() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AddConfigPath("./config")
	viper.AddConfigPath("../together-server/config")
	viper.AddConfigPath("/etc/together/")

	err := viper.ReadInConfig()

	if err != nil {
		panic(fmt.Errorf("Fatal error reading config file: %s\n", err))
	}

	viper.Unmarshal(&config)

	if config.DataDir != "" {
		engine.DataDir = config.DataDir
	}

	if config.ChunksDir != "" {
		engine.ChunksDir = config.ChunksDir
	}

	if config.WorldMaxAltitude != 0 {
		engine.WorldMaxAltitude = config.WorldMaxAltitude
	}

	if config.ChunkLoadRadius != 0 {
		engine.ChunkLoadRadius = config.ChunkLoadRadius
	}

	if config.ChunkLoadPadding != 0 {
		engine.ChunkLoadPadding = config.ChunkLoadPadding
	}

	if config.ChunkSize != 0 {
		engine.ChunkSize = config.ChunkSize
	}
}
