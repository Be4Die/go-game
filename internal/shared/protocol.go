package shared

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

type WorldStateMessage struct {
	Players    []PlayerState `json:"players"`
	ServerTime int64         `json:"serverTime"`
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
