package components

import rl "github.com/gen2brain/raylib-go/raylib"

// Константы для анимаций (индексы)
const (
	CharacterAnimIdle   = 1
	CharacterAnimWalk   = 2
	CharacterAnimSprint = 3
	CharacterAnimJump   = 4
	CharacterAnimFall   = 5
)

type Animator struct {
	Animations   []rl.ModelAnimation
	AnimCount    int32
	CurrentAnim  int32
	CurrentFrame int32
}

func (a *Animator) Mask() uint64 {
	return MaskAnimator
}

func NewAnimator(modelPath string) *Animator {
	animations := rl.LoadModelAnimations(modelPath)
	animCount := int32(len(animations))

	return &Animator{
		Animations:   animations,
		AnimCount:    animCount,
		CurrentAnim:  0,
		CurrentFrame: 0,
	}
}
