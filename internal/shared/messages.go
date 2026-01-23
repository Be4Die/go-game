package shared

import "time"

type MessageType int

const (
	MessageTypeWelcome MessageType = iota + 1
	MessageTypeJoin
	MessageTypeLeave
	MessageTypeInput
	MessageTypeWorldState
	MessageTypeHeartbeat
	MessageTypeNewPlayer
	MessageTypePlayerLeft
	MessageTypeError
)

type Message struct {
	Type     MessageType `json:"type"`
	Data     []byte      `json:"data"`
	PlayerID string      `json:"playerId,omitempty"`
}

type WelcomeMessage struct {
	PlayerID string `json:"playerId"`
	Message  string `json:"message"`
}

type JoinMessage struct {
	Nickname string `json:"nickname"`
}

type LeaveMessage struct {
	PlayerID string `json:"playerId"`
	Reason   string `json:"reason,omitempty"`
}

type InputMessage struct {
	PlayerID  string    `json:"playerId"`
	Position  Vector3   `json:"position"`
	Rotation  float32   `json:"rotation"`
	Animation int32     `json:"animation"`
	Keys      InputKeys `json:"keys"`
	Timestamp int64     `json:"timestamp"`
}

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

type WorldStateMessage struct {
	Players    []PlayerState `json:"players"`
	ServerTime int64         `json:"serverTime"`
}

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

type HeartbeatMessage struct {
	Timestamp int64 `json:"timestamp"`
}

type NewPlayerMessage struct {
	PlayerID string `json:"playerId"`
	Nickname string `json:"nickname"`
}

type PlayerLeftMessage struct {
	PlayerID string `json:"playerId"`
	Nickname string `json:"nickname"`
	Reason   string `json:"reason"`
}

type ErrorMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Vector3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

func (ps *PlayerState) Ping() {
	ps.LastUpdate = time.Now()
	ps.IsActive = true
}

func (ps *PlayerState) IsTimedOut(timeout time.Duration) bool {
	return time.Since(ps.LastUpdate) > timeout
}
