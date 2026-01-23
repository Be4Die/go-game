package components

type Remote struct {
	ID       string
	Nickname string
}

func (r *Remote) Mask() uint64 {
	return MaskRemote
}
