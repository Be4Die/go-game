package components

type Player struct {
	Speed      float32
	JumpSpeed  float32
	IsGrounded bool
	Nickname   string
}

func (p *Player) Mask() uint64 {
	return MaskPlayer
}

func NewPlayer(nickname string) *Player {
	return &Player{
		Nickname:   nickname,
		Speed:      5.0,
		JumpSpeed:  15.0,
		IsGrounded: true,
	}
}
