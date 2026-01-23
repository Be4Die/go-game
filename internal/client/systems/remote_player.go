package systems

import (
	"game/internal/client"
	"game/internal/client/components"
	"math"

	"github.com/andygeiss/ecs"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type RemotePlayerSystem struct {
	container *client.DataContainer
	entities  map[string]ecs.Entity
}

func NewRemotePlayerSystem(container *client.DataContainer) *RemotePlayerSystem {
	return &RemotePlayerSystem{
		container: container,
		entities:  make(map[string]ecs.Entity),
	}
}

func (r *RemotePlayerSystem) Process(em ecs.EntityManager) (state int) {
	if r.container.GameState != client.GameStateRunning {
		return ecs.StateEngineContinue
	}

	r.container.Mu.RLock()
	players := make(map[string]*client.RemotePlayer)
	for id, player := range r.container.Players {
		players[id] = player
	}
	r.container.Mu.RUnlock()

	// Удаляем сущности для игроков, которых больше нет в списке
	for id, entity := range r.entities {
		if _, exists := players[id]; !exists {
			em.Remove(&entity)
			delete(r.entities, id)
			continue
		}

		// Также удаляем, если игрок не активен
		if player, exists := r.container.GetPlayer(id); exists && !player.IsActive {
			em.Remove(&entity)
			delete(r.entities, id)
		}
	}

	// Добавляем/обновляем активных игроков
	for _, remotePlayer := range players {
		// Пропускаем локального игрока
		if remotePlayer.ID == r.container.PlayerID {
			continue
		}

		// Пропускаем неактивных игроков
		if !remotePlayer.IsActive {
			continue
		}

		entity, exists := r.entities[remotePlayer.ID]
		if exists {
			// Обновляем существующую сущность
			r.updateRemoteEntity(entity, remotePlayer)
		} else {
			// Создаем новую сущность
			newEntity := r.createRemoteEntity(remotePlayer)
			r.entities[remotePlayer.ID] = *newEntity
			em.Add(newEntity)
		}
	}

	return ecs.StateEngineContinue
}

func (r *RemotePlayerSystem) createRemoteEntity(player *client.RemotePlayer) *ecs.Entity {
	transform := &components.Transform{
		Position: player.Position,
		Scale:    rl.NewVector3(2, 2, 2),
	}
	transform.SetYaw(player.Rotation)

	return ecs.NewEntity("remote_"+player.ID, []ecs.Component{
		transform,
		components.NewModel("assets/characters/character-male-c.glb").
			WithTexture("assets/characters/colormap.png"),
		components.NewAnimator("assets/characters/character-male-c.glb"),
		&components.Remote{
			ID:       player.ID,
			Nickname: player.Nickname,
		},
	})
}

func (r *RemotePlayerSystem) updateRemoteEntity(entity ecs.Entity, player *client.RemotePlayer) {
	// Обновляем позицию с интерполяцией
	transform := entity.Get(components.MaskTransform)
	if transform != nil {
		t := transform.(*components.Transform)

		// Интерполяция позиции
		frameTime := rl.GetFrameTime()
		interpolationSpeed := float32(10.0)

		// Вычисляем разницу между текущей и целевой позицией
		deltaX := player.Position.X - t.Position.X
		deltaY := player.Position.Y - t.Position.Y
		deltaZ := player.Position.Z - t.Position.Z

		// Вычисляем расстояние
		distance := float32(math.Sqrt(float64(deltaX*deltaX + deltaY*deltaY + deltaZ*deltaZ)))

		if distance > 0.1 {
			// Интерполируем позицию
			moveAmount := interpolationSpeed * frameTime
			if distance < moveAmount {
				t.Position = player.Position
			} else {
				factor := moveAmount / distance
				t.Position.X += deltaX * factor
				t.Position.Y += deltaY * factor
				t.Position.Z += deltaZ * factor
			}
		} else {
			t.Position = player.Position
		}

		// Интерполяция вращения
		currentYaw := t.GetYaw()
		targetYaw := player.Rotation

		// Нормализуем углы
		for targetYaw-currentYaw > math.Pi {
			currentYaw += 2 * math.Pi
		}
		for targetYaw-currentYaw < -math.Pi {
			currentYaw -= 2 * math.Pi
		}

		rotationDiff := targetYaw - currentYaw
		rotationStep := rotationDiff * interpolationSpeed * frameTime
		t.SetYaw(currentYaw + rotationStep)
	}

	// Обновляем анимацию
	animator := entity.Get(components.MaskAnimator)
	if animator != nil {
		a := animator.(*components.Animator)
		if a.CurrentAnim != player.Animation {
			a.CurrentAnim = player.Animation
			a.CurrentFrame = 0
		}
	}
}

func (r *RemotePlayerSystem) Setup() {}
func (r *RemotePlayerSystem) Teardown() {
	for _, entity := range r.entities {
		r.container.EntityManager.Remove(&entity)
	}
	r.entities = make(map[string]ecs.Entity)
}
