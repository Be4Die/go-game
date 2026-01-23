package systems

import (
	"game/internal/client/components"
	"math"

	"github.com/andygeiss/ecs"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type InterpolationSystem struct{}

func NewInterpolationSystem() *InterpolationSystem {
	return &InterpolationSystem{}
}

func (s *InterpolationSystem) Process(em ecs.EntityManager) (state int) {
	for _, entity := range em.FilterByMask(components.MaskTransform | components.MaskInterpolation) {
		transform := entity.Get(components.MaskTransform).(*components.Transform)
		interp := entity.Get(components.MaskInterpolation).(*components.Interpolation)

		frameTime := rl.GetFrameTime()

		// Интерполяция позиции
		deltaX := interp.TargetPosition.X - transform.Position.X
		deltaY := interp.TargetPosition.Y - transform.Position.Y
		deltaZ := interp.TargetPosition.Z - transform.Position.Z

		distance := float32(math.Sqrt(float64(deltaX*deltaX + deltaY*deltaY + deltaZ*deltaZ)))

		if distance > 0.1 {
			moveAmount := interp.Speed * frameTime
			if distance < moveAmount {
				transform.Position = interp.TargetPosition
			} else {
				factor := moveAmount / distance
				transform.Position.X += deltaX * factor
				transform.Position.Y += deltaY * factor
				transform.Position.Z += deltaZ * factor
			}
		} else {
			transform.Position = interp.TargetPosition
		}

		// Интерполяция вращения
		currentYaw := transform.GetYaw()
		targetYaw := interp.TargetRotation

		for targetYaw-currentYaw > math.Pi {
			currentYaw += 2 * math.Pi
		}
		for targetYaw-currentYaw < -math.Pi {
			currentYaw -= 2 * math.Pi
		}

		rotationDiff := targetYaw - currentYaw
		rotationStep := rotationDiff * interp.Speed * frameTime
		transform.SetYaw(currentYaw + rotationStep)
	}

	return ecs.StateEngineContinue
}

func (s *InterpolationSystem) Setup()    {}
func (s *InterpolationSystem) Teardown() {}
