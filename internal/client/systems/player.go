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
	jumpVelocity      float32
	isJumping         bool
	gravity           float32
	currentYaw        float32
	targetYaw         float32
	lastInputSendTime time.Time
	inputBuffer       []shared.InputMessage
	lastServerState   *shared.PlayerState
}

func NewPlayerSystem(container *client.DataContainer) *PlayerSystem {
	return &PlayerSystem{
		container:         container,
		gravity:           20,
		jumpVelocity:      0,
		isJumping:         false,
		lastInputSendTime: time.Now(),
		inputBuffer:       make([]shared.InputMessage, 0),
	}
}

func (ps *PlayerSystem) Process(em ecs.EntityManager) (state int) {
	if ps.container.GameState == client.GameStateMenu {
		return ecs.StateEngineContinue
	}

	for _, entity := range em.FilterByMask(components.MaskPlayerController | components.MaskTransform | components.MaskAnimator) {
		player := entity.Get(components.MaskPlayerController).(*components.PlayerController)
		transform := entity.Get(components.MaskTransform).(*components.Transform)
		animator := entity.Get(components.MaskAnimator).(*components.Animator)

		// Optional: Update Identity nickname if needed, but it's usually static
		// identity := entity.Get(components.MaskNetworkIdentity).(*components.NetworkIdentity)

		isMoving := ps.processMovement(player, transform)
		ps.processJump(player, transform)
		ps.updateAnimationState(animator, isMoving)
		ps.updateCameraPosition(transform)

		// Отправляем ввод на сервер
		ps.sendInputToServer(transform, animator)

		// Применяем предикцию
		ps.applyPrediction(transform)
	}

	return ecs.StateEngineContinue
}

func (ps *PlayerSystem) sendInputToServer(transform *components.Transform, animator *components.Animator) {
	if ps.container.Network == nil || !ps.container.Network.IsConnected() {
		return
	}

	now := time.Now()
	if now.Sub(ps.lastInputSendTime) < 16*time.Millisecond {
		return
	}

	// Собираем ввод
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
		Timestamp: now.UnixNano() / int64(time.Millisecond),
	}

	// Сохраняем в буфер для предикции
	ps.inputBuffer = append(ps.inputBuffer, inputMsg)
	if len(ps.inputBuffer) > 60 {
		ps.inputBuffer = ps.inputBuffer[1:]
	}

	// Отправляем на сервер
	if err := ps.container.Network.SendInput(inputMsg); err != nil {
		// Обработка ошибки
	}

	ps.lastInputSendTime = now
}

func (ps *PlayerSystem) applyPrediction(transform *components.Transform) {
	// Предикция - предсказываем положение на основе последнего известного состояния сервера
	// и буфера ввода. Это упрощенная версия.
	// Здесь можно добавить логику согласования с сервером
}

func (ps *PlayerSystem) processMovement(player *components.PlayerController, transform *components.Transform) bool {
	moveDirection := ps.calculateMoveDirection()
	isMoving := moveDirection.X != 0 || moveDirection.Z != 0

	if !isMoving {
		return false
	}

	speed := ps.calculateMovementSpeed(player.Speed, isMoving)

	// Нормализуем вектор направления
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
	transform.Position.X += direction.X * speed * frameTime
	transform.Position.Z += direction.Z * speed * frameTime
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

func (ps *PlayerSystem) Setup() {
	playerEntity := ecs.NewEntity("player", []ecs.Component{
		&components.Transform{
			Position: rl.NewVector3(0.0, 0.0, 0.0),
			Rotation: rl.Quaternion{X: 0, Y: 0, Z: 0, W: 1},
			Scale:    rl.NewVector3(2, 2, 2),
		},
		components.NewModel("assets/characters/character-male-c.glb").
			WithTexture("assets/characters/colormap.png"),
		components.NewAnimator("assets/characters/character-male-c.glb"),
		components.NewPlayerController(),
		&components.NetworkIdentity{
			ID:       ps.container.PlayerID,
			Nickname: ps.container.PlayerNickname,
			IsLocal:  true,
		},
	})

	ps.container.EntityManager.Add(playerEntity)
}

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
