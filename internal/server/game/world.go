package game

import (
	"game/internal/shared"
	"math"
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
		PlayerID:    id,
		Nickname:    nickname,
		Model:       model,
		Position:    shared.Vector3{X: 0, Y: 0, Z: 0},
		Rotation:    0,
		Velocity:    shared.Vector3{X: 0, Y: 0, Z: 0},
		Animation:   1,
		IsGrounded:  true,
		IsJumping:   false,
		JoinedAt:    time.Now(),
		LastUpdate:  time.Now(),
		IsActive:    true,
		InputBuffer: make([]shared.InputMessage, 0),
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

	// Buffer the input instead of overwriting state immediately
	player.InputBuffer = append(player.InputBuffer, input)
}

func (w *World) Update(deltaTime float32) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, player := range w.players {
		if !player.IsActive {
			continue
		}

		// Process all buffered inputs
		if len(player.InputBuffer) > 0 {
			for _, input := range player.InputBuffer {
				// Apply state from input
				player.Rotation = input.Rotation
				player.Animation = input.Animation
				player.LastInput = input.Keys // Store for debug/reference

				// Use client-provided delta time for precise synchronization
				// Clamp to reasonable limits to prevent speed hacking or huge jumps
				dt := input.DeltaTime
				if dt > 0.1 { // Max 100ms per packet to prevent teleportation
					dt = 0.1
				}
				if dt <= 0 { // Fallback if something is wrong
					dt = 0.016
				}

				// Apply physics for this input slice
				w.applyPhysics(player, input.Keys, dt)
			}
			// Clear buffer after processing
			player.InputBuffer = player.InputBuffer[:0]
		}
	}
}

func (w *World) applyPhysics(player *shared.PlayerState, input shared.InputKeys, deltaTime float32) {
	speed := float32(5.0)
	if input.Sprint {
		speed = 10.0
	}

	// Calculate movement vector
	moveX := float32(0.0)
	moveZ := float32(0.0)

	if input.Forward {
		moveZ -= 1
	}
	if input.Backward {
		moveZ += 1
	}
	if input.Left {
		moveX -= 1
	}
	if input.Right {
		moveX += 1
	}

	// Normalize vector to prevent diagonal speed boost
	length := float32(math.Sqrt(float64(moveX*moveX + moveZ*moveZ)))
	if length > 0 {
		moveX /= length
		moveZ /= length
	}

	// Store horizontal velocity (units/sec) for client-side extrapolation
	if length > 0 {
		player.Velocity.X = moveX * speed
		player.Velocity.Z = moveZ * speed
	} else {
		player.Velocity.X = 0
		player.Velocity.Z = 0
	}

	// Apply movement
	player.Position.X += moveX * speed * deltaTime
	player.Position.Z += moveZ * speed * deltaTime

	// Jump Logic
	if input.Jump && !player.IsJumping && player.IsGrounded {
		player.Velocity.Y = 10.0
		player.IsJumping = true
		player.IsGrounded = false // Immediately un-ground to prevent double jumps in same tick
	}

	// Gravity and Vertical Physics
	player.Velocity.Y -= w.gravity * deltaTime
	player.Position.Y += player.Velocity.Y * deltaTime

	// Ground Collision
	if player.Position.Y < 0 {
		player.Position.Y = 0
		player.Velocity.Y = 0
		player.IsJumping = false
		player.IsGrounded = true
	} else {
		// If above ground, we are not grounded (unless we just jumped)
		if player.Position.Y > 0 {
			player.IsGrounded = false
		}
	}

	player.LastUpdate = time.Now()

	// Update Animation State based on input (server authoritative animation state)
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
