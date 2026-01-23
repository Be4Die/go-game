package components

type NetworkIdentity struct {
	ID       string
	Nickname string
	Model    string
	IsLocal  bool
}

func (n *NetworkIdentity) Mask() uint64 {
	return MaskNetworkIdentity
}
