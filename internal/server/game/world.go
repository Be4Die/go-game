package game

import (
	"game/internal/shared"
	"sync"
	"time"
)

type World struct {
	players map[string]*shared.PlayerState
	mu      sync.RWMutex
	gravity float32
}

func NewWorld() *World {
	return &World{
		players: make(map[string]*shared.PlayerState),
		gravity: 20.0,
	}
}

func (w *World) AddPlayer(id, nickname, model string) *shared.PlayerState {
	w.mu.Lock()
	defer w.mu.Unlock()

	player := &shared.PlayerState{
		PlayerID:   id,
		Nickname:   nickname,
		Model:      model,
		Position:   shared.Vector3{X: 0, Y: 0, Z: 0},
		Rotation:   0,
		Velocity:   shared.Vector3{X: 0, Y: 0, Z: 0},
		Animation:  1,
		IsGrounded: true,
		IsJumping:  false,
		JoinedAt:   time.Now(),
		LastUpdate: time.Now(),
		IsActive:   true,
	}
	w.players[id] = player
	return player
}

func (w *World) RemovePlayer(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.players, id)
}

func (w *World) GetPlayer(id string) *shared.PlayerState {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.players[id]
}

func (w *World) GetAllPlayers() []shared.PlayerState {
	w.mu.RLock()
	defer w.mu.RUnlock()

	players := make([]shared.PlayerState, 0, len(w.players))
	for _, p := range w.players {
		players = append(players, *p)
	}
	return players
}

func (w *World) ProcessInput(playerID string, input shared.InputMessage) {
	w.mu.Lock()
	defer w.mu.Unlock()

	player, ok := w.players[playerID]
	if !ok {
		return
	}

	player.LastInput = input.Keys
	player.Rotation = input.Rotation
	player.Animation = input.Animation
	// In authoritative server, we might validate position, but here we trust input rotation/anim
}

func (w *World) Update(deltaTime float32) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, player := range w.players {
		if !player.IsActive {
			continue
		}

		if !player.LastInput.IsEmpty() {
			w.applyPhysics(player, player.LastInput, deltaTime)
			player.LastInput = shared.InputKeys{}
		}
	}
}

func (w *World) applyPhysics(player *shared.PlayerState, input shared.InputKeys, deltaTime float32) {
	speed := float32(5.0)
	if input.Sprint {
		speed = 10.0
	}

	if input.Forward {
		player.Position.Z -= speed * deltaTime
	}
	if input.Backward {
		player.Position.Z += speed * deltaTime
	}
	if input.Left {
		player.Position.X -= speed * deltaTime
	}
	if input.Right {
		player.Position.X += speed * deltaTime
	}

	if input.Jump && !player.IsJumping && player.IsGrounded {
		player.Velocity.Y = 10.0
		player.IsJumping = true
	}

	player.Velocity.Y -= w.gravity * deltaTime
	player.Position.Y += player.Velocity.Y * deltaTime

	if player.Position.Y < 0 {
		player.Position.Y = 0
		player.Velocity.Y = 0
		player.IsJumping = false
		player.IsGrounded = true
	}

	player.LastUpdate = time.Now()

	if input.Forward || input.Backward || input.Left || input.Right {
		player.Animation = 2 // Walk
		if input.Sprint {
			player.Animation = 3 // Run
		}
	} else {
		player.Animation = 1 // Idle
	}

	if player.IsJumping {
		player.Animation = 4 // Jump
	}
}
