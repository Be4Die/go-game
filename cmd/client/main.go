package main

import (
	"game/internal/client"
	"game/internal/client/systems"
	"game/internal/shared"

	"github.com/andygeiss/ecs"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func main() {
	camera := rl.Camera3D{
		Position:   rl.NewVector3(0.0, 20.0, 10.0),
		Target:     rl.NewVector3(0.0, 0.0, 0.0),
		Up:         rl.NewVector3(0.0, 1.0, 0.0),
		Fovy:       45.0,
		Projection: rl.CameraPerspective,
	}

	em := ecs.NewEntityManager()
	sm := ecs.NewSystemManager()

	container := client.DataContainer{
		GameState:     client.GameStateMenu,
		Camera:        &camera,
		EntityManager: em,
		SystemManager: sm,
		Players:       make(map[string]*shared.PlayerState),
	}

	sm.Add(systems.NewRenderingSystem(&container))
	sm.Add(systems.NewPlayerSystem(&container))
	sm.Add(systems.NewAnimationSystem())
	sm.Add(systems.NewRemotePlayerSystem(&container))
	sm.Add(systems.NewInterpolationSystem())

	de := ecs.NewDefaultEngine(em, sm)
	de.Setup()
	defer de.Teardown()
	de.Run()
}
