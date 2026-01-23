package shared

type InputKeys struct {
	Forward  bool `json:"forward"`
	Backward bool `json:"backward"`
	Left     bool `json:"left"`
	Right    bool `json:"right"`
	Jump     bool `json:"jump"`
	Sprint   bool `json:"sprint"`
}

func (ik InputKeys) IsEmpty() bool {
	return !ik.Forward && !ik.Backward && !ik.Left && !ik.Right && !ik.Jump && !ik.Sprint
}
