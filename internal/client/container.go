package client

import (
	"game/internal/shared"
	"sync"
	"time"

	"github.com/andygeiss/ecs"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	GameStateMenu = iota
	GameStateRunning
	GameStateConnecting
	GameStateError
)

type DataContainer struct {
	GameState      int
	PlayerNickname string
	PlayerModel    string
	PlayerID       string
	Camera         *rl.Camera
	EntityManager  ecs.EntityManager
	SystemManager  ecs.SystemManager
	Network        *Network
	Players        map[string]*shared.PlayerState
	NetworkError   string
	Mu             sync.RWMutex

	// Статистика сети
	LastPacketTime time.Time
	PacketCount    int64
}

// AddPlayer добавляет удаленного игрока
func (c *DataContainer) AddPlayer(player *shared.PlayerState) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.Players[player.PlayerID] = player
}

// RemovePlayer удаляет удаленного игрока
func (c *DataContainer) RemovePlayer(playerID string) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	delete(c.Players, playerID)
}

// GetPlayer возвращает удаленного игрока
func (c *DataContainer) GetPlayer(playerID string) (*shared.PlayerState, bool) {
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	player, exists := c.Players[playerID]
	return player, exists
}

// GetAllPlayers возвращает всех удаленных игроков
func (c *DataContainer) GetAllPlayers() []*shared.PlayerState {
	c.Mu.RLock()
	defer c.Mu.RUnlock()

	players := make([]*shared.PlayerState, 0, len(c.Players))
	for _, player := range c.Players {
		players = append(players, player)
	}
	return players
}

// ClearPlayers очищает всех удаленных игроков
func (c *DataContainer) ClearPlayers() {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.Players = make(map[string]*shared.PlayerState)
}

// SetNetworkError устанавливает ошибку сети
func (c *DataContainer) SetNetworkError(err string) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.NetworkError = err
	c.GameState = GameStateError
}

// ClearNetworkError очищает ошибку сети
func (c *DataContainer) ClearNetworkError() {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.NetworkError = ""
	if c.GameState == GameStateError {
		c.GameState = GameStateMenu
	}
}
