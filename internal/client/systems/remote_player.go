package systems

import (
	"game/internal/client"
	"game/internal/client/components"
	"game/internal/shared"

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
	players := make(map[string]*shared.PlayerState)
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

		if player, exists := r.container.GetPlayer(id); exists && !player.IsActive {
			em.Remove(&entity)
			delete(r.entities, id)
		}
	}

	// Добавляем/обновляем активных игроков
	for _, remotePlayer := range players {
		// Пропускаем локального игрока
		if remotePlayer.PlayerID == r.container.PlayerID {
			continue
		}

		// Пропускаем неактивных игроков
		if !remotePlayer.IsActive {
			continue
		}

		entity, exists := r.entities[remotePlayer.PlayerID]
		if exists {
			// Обновляем существующую сущность
			r.updateRemoteEntity(entity, remotePlayer)
		} else {
			// Создаем новую сущность
			newEntity := r.createRemoteEntity(remotePlayer)
			r.entities[remotePlayer.PlayerID] = *newEntity
			em.Add(newEntity)
		}
	}

	return ecs.StateEngineContinue
}

func (r *RemotePlayerSystem) createRemoteEntity(player *shared.PlayerState) *ecs.Entity {
	pos := rl.NewVector3(player.Position.X, player.Position.Y, player.Position.Z)

	transform := &components.Transform{
		Position: pos,
		Scale:    rl.NewVector3(2, 2, 2),
	}
	transform.SetYaw(player.Rotation)

	return ecs.NewEntity("remote_"+player.PlayerID, []ecs.Component{
		transform,
		components.NewModel("assets/characters/character-male-c.glb").
			WithTexture("assets/characters/colormap.png"),
		components.NewAnimator("assets/characters/character-male-c.glb"),
		&components.NetworkIdentity{
			ID:       player.PlayerID,
			Nickname: player.Nickname,
			IsLocal:  false,
		},
		&components.Interpolation{
			TargetPosition: pos,
			TargetRotation: player.Rotation,
			Speed:          10.0,
		},
	})
}

func (r *RemotePlayerSystem) updateRemoteEntity(entity ecs.Entity, player *shared.PlayerState) {
	// Обновляем цель интерполяции
	interp := entity.Get(components.MaskInterpolation)
	if interp != nil {
		i := interp.(*components.Interpolation)
		i.TargetPosition = rl.NewVector3(player.Position.X, player.Position.Y, player.Position.Z)
		i.TargetRotation = player.Rotation
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
