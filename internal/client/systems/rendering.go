package systems

import (
	"game/internal/client"
	"game/internal/client/rendering"

	"github.com/andygeiss/ecs"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type RenderingSystem struct {
	container    *client.DataContainer
	gameRenderer *rendering.GameRenderer
	menuRenderer *rendering.MenuRenderer
}

func NewRenderingSystem(container *client.DataContainer) *RenderingSystem {
	return &RenderingSystem{
		container: container,
	}
}

func (r *RenderingSystem) Setup() {
	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(1280, 720, "client")
	rl.SetTargetFPS(60)

	// Initialize both renderers
	r.gameRenderer = rendering.NewGameRenderer(r.container.Camera)
	r.menuRenderer = rendering.NewMenuRenderer(r.container)

	r.gameRenderer.Setup()
	r.menuRenderer.Setup()
}

func (r *RenderingSystem) Process(em ecs.EntityManager) (state int) {
	if rl.WindowShouldClose() {
		return ecs.StateEngineStop
	}

	rl.BeginDrawing()
	rl.ClearBackground(rl.RayWhite)

	switch r.container.GameState {

	case client.GameStateMenu:
		r.menuRenderer.Process()

	case client.GameStateRunning:
		if r.container.WorldSeed != 0 && !r.gameRenderer.IsWorldLoaded() {
			r.gameRenderer.LoadWorld(r.container.WorldSeed)
		}
		r.gameRenderer.Process(em)

	}

	rl.EndDrawing()

	return ecs.StateEngineContinue
}

func (r *RenderingSystem) Teardown() {
	if r.gameRenderer != nil {
		r.gameRenderer.Teardown()
	}
	if r.menuRenderer != nil {
		r.menuRenderer.Teardown()
	}
	rl.CloseWindow()
}
