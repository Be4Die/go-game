package systems

import (
	"game/internal/client/components"

	"github.com/andygeiss/ecs"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type AnimationSystem struct{}

func NewAnimationSystem() *AnimationSystem {
	return &AnimationSystem{}
}

func (a *AnimationSystem) Process(em ecs.EntityManager) (state int) {
	for _, e := range em.FilterByMask(components.MaskAnimator | components.MaskModel) {
		animator := e.Get(components.MaskAnimator).(*components.Animator)
		model := e.Get(components.MaskModel).(*components.Model)

		if animator.AnimCount == 0 || animator.CurrentAnim >= animator.AnimCount {
			continue
		}

		// Обновляем текущую анимацию
		rl.UpdateModelAnimation(
			model.Model,
			animator.Animations[animator.CurrentAnim],
			animator.CurrentFrame,
		)

		animator.CurrentFrame++

		// Сбрасываем кадр если достигли конца анимации
		if animator.CurrentFrame >= animator.Animations[animator.CurrentAnim].FrameCount {
			animator.CurrentFrame = 0
		}
	}

	return ecs.StateEngineContinue
}

func (a *AnimationSystem) Setup()    {}
func (a *AnimationSystem) Teardown() {}
