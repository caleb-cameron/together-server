package main

import (
	"fmt"
	"strconv"
	"time"

	engine "github.com/abeardevil/together-engine"
	perlin "github.com/aquilax/go-perlin"
)

func initServer() {
	engine.InitTiles()
	engine.GWorld = engine.World{}
	engine.GWorld.Init()

	setWorldSeed()

	engine.GWorld.Generate()
	engine.GWorld.GenerateBoundaryTiles()
}

func setWorldSeed() {
	var err error
	seed := engine.GWorld.GetSeed()

	if seed == 0 {
		if config.WorldSeed != "" {
			seed, err = strconv.ParseUint(config.WorldSeed, 10, 64)
			if err != nil {
				panic(fmt.Errorf("Bad WorldSeed value: %s\n", config.WorldSeed))
			}
		} else {
			seed = uint64(time.Now().UnixNano())
		}
		engine.GWorld.SetSeed(seed)
	}

	engine.Noise = perlin.NewPerlin(2.0, 2.0, 3, int64(seed))
}
