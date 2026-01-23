package components

type PlayerController struct {
	Speed      float32
	JumpSpeed  float32
	IsGrounded bool
}

func (p *PlayerController) Mask() uint64 {
	return MaskPlayerController
}

func NewPlayerController() *PlayerController {
	return &PlayerController{
		Speed:      5.0,
		JumpSpeed:  15.0,
		IsGrounded: true,
	}
}
