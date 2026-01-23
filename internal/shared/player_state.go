package shared

import "time"

type PlayerState struct {
	PlayerID   string    `json:"playerId"`
	Nickname   string    `json:"nickname"`
	Position   Vector3   `json:"position"`
	Rotation   float32   `json:"rotation"`
	Animation  int32     `json:"animation"`
	Velocity   Vector3   `json:"velocity"`
	IsGrounded bool      `json:"isGrounded"`
	IsJumping  bool      `json:"isJumping"`
	LastInput  InputKeys `json:"-"`
	JoinedAt   time.Time `json:"joinedAt"`
	LastUpdate time.Time `json:"lastUpdate"`
	IsActive   bool      `json:"isActive"`
}

func (ps *PlayerState) Ping() {
	ps.LastUpdate = time.Now()
	ps.IsActive = true
}

func (ps *PlayerState) IsTimedOut(timeout time.Duration) bool {
	return time.Since(ps.LastUpdate) > timeout
}
