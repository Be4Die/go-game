package components

import rl "github.com/gen2brain/raylib-go/raylib"

type Interpolation struct {
	TargetPosition rl.Vector3
	TargetRotation float32
	Speed          float32
}

func (i *Interpolation) Mask() uint64 {
	return MaskInterpolation
}
