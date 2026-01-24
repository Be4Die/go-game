package systems

import (
	"game/internal/client"
	"game/internal/client/components"
	"game/internal/shared"
	"math"
	"time"

	"github.com/andygeiss/ecs"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type PlayerSystem struct {
	container         *client.DataContainer
	spawned           bool
	jumpVelocity      float32
	isJumping         bool
	gravity           float32
	currentYaw        float32
	targetYaw         float32
	lastInputSendTime time.Time
	accumulatedTime   float32
	inputBuffer       []shared.InputMessage
	lastServerState   *shared.PlayerState
	worldSeed         int64
	chunks            map[int64][]shared.StaticObject
	centerChunkX      int32
	centerChunkZ      int32
	collisionRadius   int32
}

func NewPlayerSystem(container *client.DataContainer) *PlayerSystem {
	return &PlayerSystem{
		container:         container,
		gravity:           20,
		jumpVelocity:      0,
		isJumping:         false,
		lastInputSendTime: time.Now(),
		accumulatedTime:   0,
		inputBuffer:       make([]shared.InputMessage, 0),
		chunks:            make(map[int64][]shared.StaticObject),
		collisionRadius:   2,
	}
}

func (ps *PlayerSystem) Process(em ecs.EntityManager) (state int) {
	if ps.container.GameState != client.GameStateRunning {
		ps.spawned = false
		return ecs.StateEngineContinue
	}

	if ps.container.WorldSeed != 0 && ps.worldSeed != ps.container.WorldSeed {
		ps.worldSeed = ps.container.WorldSeed
		ps.chunks = make(map[int64][]shared.StaticObject)
		ps.centerChunkX = 0
		ps.centerChunkZ = 0
	}

	if !ps.spawned {
		ps.spawnLocalPlayer(em)
		ps.spawned = true
	}

	// Accumulate time for input synchronization
	ps.accumulatedTime += rl.GetFrameTime()

	for _, entity := range em.FilterByMask(components.MaskPlayerController | components.MaskTransform | components.MaskAnimator) {
		player := entity.Get(components.MaskPlayerController).(*components.PlayerController)
		transform := entity.Get(components.MaskTransform).(*components.Transform)
		animator := entity.Get(components.MaskAnimator).(*components.Animator)

		isMoving := ps.processMovement(player, transform)
		ps.processJump(player, transform)
		ps.updateAnimationState(animator, isMoving)
		ps.updateCameraPosition(transform)

		// Send input to server
		ps.sendInputToServer(transform, animator)

		// Apply prediction
		ps.applyPrediction(transform)
	}

	return ecs.StateEngineContinue
}

func (ps *PlayerSystem) spawnLocalPlayer(em ecs.EntityManager) {
	for _, entity := range em.FilterByMask(components.MaskNetworkIdentity) {
		identity, ok := entity.Get(components.MaskNetworkIdentity).(*components.NetworkIdentity)
		if !ok || identity == nil {
			continue
		}
		if identity.IsLocal {
			em.Remove(entity)
		}
	}

	modelPath := "assets/characters/character-male-c.glb"
	if ps.container.PlayerModel != "" {
		modelPath = ps.container.PlayerModel
	}

	playerEntity := ecs.NewEntity("player", []ecs.Component{
		&components.Transform{
			Position: rl.NewVector3(0.0, 0.0, 0.0),
			Rotation: rl.Quaternion{X: 0, Y: 0, Z: 0, W: 1},
			Scale:    rl.NewVector3(2, 2, 2),
		},
		components.NewModel(modelPath).
			WithTexture("assets/characters/colormap.png"),
		components.NewAnimator(modelPath),
		components.NewPlayerController(),
		&components.NetworkIdentity{
			ID:       ps.container.PlayerID,
			Nickname: ps.container.PlayerNickname,
			Model:    ps.container.PlayerModel,
			IsLocal:  true,
		},
	})

	em.Add(playerEntity)
}

func (ps *PlayerSystem) sendInputToServer(transform *components.Transform, animator *components.Animator) {
	if ps.container.Network == nil || !ps.container.Network.IsConnected() {
		return
	}

	now := time.Now()
	// Send input approx every 16ms (60hz), but include exact accumulated time
	if now.Sub(ps.lastInputSendTime) < 16*time.Millisecond {
		return
	}

	// Collect input
	inputKeys := shared.InputKeys{
		Forward:  rl.IsKeyDown(rl.KeyW),
		Backward: rl.IsKeyDown(rl.KeyS),
		Left:     rl.IsKeyDown(rl.KeyA),
		Right:    rl.IsKeyDown(rl.KeyD),
		Jump:     rl.IsKeyPressed(rl.KeySpace),
		Sprint:   rl.IsKeyDown(rl.KeyLeftShift),
	}

	inputMsg := shared.InputMessage{
		PlayerID:  ps.container.PlayerID,
		Position:  shared.Vector3{X: transform.Position.X, Y: transform.Position.Y, Z: transform.Position.Z},
		Rotation:  transform.GetYaw(),
		Animation: animator.CurrentAnim,
		Keys:      inputKeys,
		DeltaTime: ps.accumulatedTime, // Send the exact time elapsed since last packet
		Timestamp: now.UnixNano() / int64(time.Millisecond),
	}

	// Reset accumulated time after sending
	ps.accumulatedTime = 0

	// Store in buffer for prediction
	ps.inputBuffer = append(ps.inputBuffer, inputMsg)
	if len(ps.inputBuffer) > 60 {
		ps.inputBuffer = ps.inputBuffer[1:]
	}

	// Send to server
	if err := ps.container.Network.SendInput(inputMsg); err != nil {
		// Handle error
	}

	ps.lastInputSendTime = now
}

func (ps *PlayerSystem) applyPrediction(transform *components.Transform) {
	// Prediction logic placeholder
}

func (ps *PlayerSystem) processMovement(player *components.PlayerController, transform *components.Transform) bool {
	moveDirection := ps.calculateMoveDirection()
	isMoving := moveDirection.X != 0 || moveDirection.Z != 0

	if !isMoving {
		return false
	}

	speed := ps.calculateMovementSpeed(player.Speed, isMoving)

	// Normalize direction vector
	length := float32(math.Sqrt(float64(moveDirection.X*moveDirection.X + moveDirection.Y*moveDirection.Y + moveDirection.Z*moveDirection.Z)))
	if length > 0 {
		moveDirection.X /= length
		moveDirection.Y /= length
		moveDirection.Z /= length
	}

	ps.updatePlayerRotation(transform, moveDirection)
	ps.applyMovement(transform, moveDirection, speed)

	return true
}

func (ps *PlayerSystem) calculateMoveDirection() rl.Vector3 {
	direction := rl.NewVector3(0, 0, 0)

	if rl.IsKeyDown(rl.KeyW) {
		direction.Z -= 1
	}
	if rl.IsKeyDown(rl.KeyS) {
		direction.Z += 1
	}
	if rl.IsKeyDown(rl.KeyA) {
		direction.X -= 1
	}
	if rl.IsKeyDown(rl.KeyD) {
		direction.X += 1
	}

	return direction
}

func (ps *PlayerSystem) calculateMovementSpeed(baseSpeed float32, isMoving bool) float32 {
	speed := baseSpeed

	if rl.IsKeyDown(rl.KeyLeftShift) && isMoving {
		speed *= 2
	}

	return speed
}

func (ps *PlayerSystem) applyMovement(transform *components.Transform, direction rl.Vector3, speed float32) {
	frameTime := rl.GetFrameTime()

	if ps.worldSeed != 0 {
		cx := shared.ChunkCoord(transform.Position.X)
		cz := shared.ChunkCoord(transform.Position.Z)
		if cx != ps.centerChunkX || cz != ps.centerChunkZ {
			ps.centerChunkX = cx
			ps.centerChunkZ = cz
			ps.ensureChunksLoaded()
		}
		if len(ps.chunks) == 0 {
			ps.ensureChunksLoaded()
		}
	}

	nextX := transform.Position.X + direction.X*speed*frameTime
	nextZ := transform.Position.Z + direction.Z*speed*frameTime

	playerRadius := float32(0.5)

	if !ps.collides(nextX, nextZ, playerRadius) {
		transform.Position.X = nextX
		transform.Position.Z = nextZ
		return
	}

	if !ps.collides(nextX, transform.Position.Z, playerRadius) {
		transform.Position.X = nextX
		return
	}

	if !ps.collides(transform.Position.X, nextZ, playerRadius) {
		transform.Position.Z = nextZ
		return
	}
}

func (ps *PlayerSystem) ensureChunksLoaded() {
	required := make(map[int64]struct{})
	for dx := -ps.collisionRadius; dx <= ps.collisionRadius; dx++ {
		for dz := -ps.collisionRadius; dz <= ps.collisionRadius; dz++ {
			cx := ps.centerChunkX + dx
			cz := ps.centerChunkZ + dz
			k := shared.ChunkKey(cx, cz)
			required[k] = struct{}{}
			if _, ok := ps.chunks[k]; !ok {
				ps.chunks[k] = shared.GenerateChunk(ps.worldSeed, cx, cz)
			}
		}
	}

	for k := range ps.chunks {
		if _, ok := required[k]; !ok {
			delete(ps.chunks, k)
		}
	}
}

func (ps *PlayerSystem) collides(x, z, radius float32) bool {
	if len(ps.chunks) == 0 {
		return false
	}

	cx := shared.ChunkCoord(x)
	cz := shared.ChunkCoord(z)

	for dx := int32(-1); dx <= 1; dx++ {
		for dz := int32(-1); dz <= 1; dz++ {
			k := shared.ChunkKey(cx+dx, cz+dz)
			objs, ok := ps.chunks[k]
			if !ok {
				continue
			}
			for _, obj := range objs {
				if shared.CollidesPointWithStaticObjectXZ(obj, x, z, radius) {
					return true
				}
			}
		}
	}

	return false
}

func (ps *PlayerSystem) updatePlayerRotation(transform *components.Transform, direction rl.Vector3) {
	ps.targetYaw = float32(math.Atan2(float64(-direction.X), float64(direction.Z)))
	rotationSpeed := float32(10.0) * rl.GetFrameTime()

	angleDiff := ps.targetYaw - ps.currentYaw
	angleDiff = ps.normalizeAngle(angleDiff)

	ps.currentYaw += angleDiff * rotationSpeed
	transform.SetYaw(ps.currentYaw)
}

func (ps *PlayerSystem) normalizeAngle(angle float32) float32 {
	if angle > math.Pi {
		angle -= 2 * math.Pi
	}
	if angle < -math.Pi {
		angle += 2 * math.Pi
	}
	return angle
}

func (ps *PlayerSystem) processJump(player *components.PlayerController, transform *components.Transform) {
	ps.handleJumpInput(player)
	ps.updateJumpPhysics(transform)
	ps.updateGroundedState(player, transform)
}

func (ps *PlayerSystem) handleJumpInput(player *components.PlayerController) {
	if rl.IsKeyPressed(rl.KeySpace) && !ps.isJumping && player.IsGrounded {
		ps.isJumping = true
		ps.jumpVelocity = 10.0
	}
}

func (ps *PlayerSystem) updateJumpPhysics(transform *components.Transform) {
	if !ps.isJumping {
		return
	}

	ps.jumpVelocity -= ps.gravity * rl.GetFrameTime()
	transform.Position.Y += ps.jumpVelocity * rl.GetFrameTime()

	if transform.Position.Y <= 0 {
		transform.Position.Y = 0
		ps.isJumping = false
		ps.jumpVelocity = 0
	}
}

func (ps *PlayerSystem) updateGroundedState(player *components.PlayerController, transform *components.Transform) {
	player.IsGrounded = !ps.isJumping && transform.Position.Y == 0
}

func (ps *PlayerSystem) updateAnimationState(animator *components.Animator, isMoving bool) {
	if animator == nil {
		return
	}

	animationIndex := ps.determineAnimationIndex(isMoving)

	if animationIndex >= animator.AnimCount {
		animationIndex = 0
	}

	if animator.CurrentAnim != animationIndex {
		animator.CurrentFrame = 0
		animator.CurrentAnim = animationIndex
	}
}

func (ps *PlayerSystem) determineAnimationIndex(isMoving bool) int32 {
	if ps.isJumping {
		if ps.jumpVelocity > 0 {
			return components.CharacterAnimJump
		}
		return components.CharacterAnimFall
	}

	if rl.IsKeyDown(rl.KeyLeftShift) && isMoving {
		return components.CharacterAnimSprint
	}

	if isMoving {
		return components.CharacterAnimWalk
	}

	return components.CharacterAnimIdle
}

func (ps *PlayerSystem) updateCameraPosition(transform *components.Transform) {
	ps.container.Camera.Position = rl.NewVector3(
		transform.Position.X,
		transform.Position.Y+5.0,
		transform.Position.Z+10.0,
	)
	ps.container.Camera.Target = rl.NewVector3(
		transform.Position.X,
		transform.Position.Y+2.0,
		transform.Position.Z,
	)
}

func (ps *PlayerSystem) Setup() {}

func (ps *PlayerSystem) Teardown() {
	for _, entity := range ps.container.EntityManager.FilterByMask(components.MaskAnimator) {
		animator, ok := entity.Get(components.MaskAnimator).(*components.Animator)
		if !ok || animator == nil {
			continue
		}

		for i := int32(0); i < animator.AnimCount; i++ {
			rl.UnloadModelAnimation(animator.Animations[i])
		}
	}
}
