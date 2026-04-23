package signaling

import (
	"context"
	"sync"
	"time"
)

type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
}

type Server struct {
	cfg    Config
	logger Logger
	rooms  map[string]*Room
	mu     sync.RWMutex
}

type Config struct {
	Port int
}

type Room struct {
	ID        string
	Name      string
	OwnerID   string
	Members   map[string]*Peer
	CreatedAt time.Time
	mu        sync.RWMutex
}

type Peer struct {
	ID        string
	UserID    string
	RoomID    string
	Connected bool
	JoinedAt  time.Time
}

func NewServer(cfg Config, logger Logger) *Server {
	return &Server{
		cfg:    cfg,
		logger: logger,
		rooms:  make(map[string]*Room),
	}
}

func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("Signaling server starting on port %d", s.cfg.Port)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				s.cleanupRooms()
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
	return nil
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, room := range s.rooms {
		room.mu.Lock()
		for _, peer := range room.Members {
			peer.Connected = false
		}
		room.mu.Unlock()
	}
	return nil
}

func (s *Server) CreateRoom(roomID, name, ownerID string) *Room {
	s.mu.Lock()
	defer s.mu.Unlock()

	room := &Room{
		ID:        roomID,
		Name:      name,
		OwnerID:   ownerID,
		Members:   make(map[string]*Peer),
		CreatedAt: time.Now(),
	}
	s.rooms[roomID] = room

	s.logger.Info("Room %s created by %s", roomID, ownerID)
	return room
}

func (s *Server) GetRoom(roomID string) (*Room, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	room, ok := s.rooms[roomID]
	return room, ok
}

func (s *Server) DeleteRoom(roomID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room, ok := s.rooms[roomID]; ok {
		room.mu.Lock()
		for _, peer := range room.Members {
			peer.Connected = false
		}
		room.mu.Unlock()
		delete(s.rooms, roomID)
		s.logger.Info("Room %s deleted", roomID)
	}
}

func (s *Server) JoinRoom(roomID, peerID, userID string) (*Peer, bool) {
	s.mu.RLock()
	room, ok := s.rooms[roomID]
	s.mu.RUnlock()

	if !ok {
		return nil, false
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	peer := &Peer{
		ID:        peerID,
		UserID:    userID,
		RoomID:    roomID,
		Connected: true,
		JoinedAt:  time.Now(),
	}
	room.Members[peerID] = peer

	s.logger.Info("Peer %s (user %s) joined room %s", peerID, userID, roomID)
	return peer, true
}

func (s *Server) LeaveRoom(roomID, peerID string) {
	s.mu.RLock()
	room, ok := s.rooms[roomID]
	s.mu.RUnlock()

	if !ok {
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if peer, ok := room.Members[peerID]; ok {
		peer.Connected = false
		delete(room.Members, peerID)
		s.logger.Info("Peer %s left room %s", peerID, roomID)
	}
}

func (s *Server) GetPeers(roomID string) []*Peer {
	s.mu.RLock()
	room, ok := s.rooms[roomID]
	s.mu.RUnlock()

	if !ok {
		return nil
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	peers := make([]*Peer, 0, len(room.Members))
	for _, peer := range room.Members {
		peers = append(peers, peer)
	}
	return peers
}

func (s *Server) cleanupRooms() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for roomID, room := range s.rooms {
		room.mu.Lock()
		hasConnected := false
		for _, peer := range room.Members {
			if peer.Connected {
				hasConnected = true
				break
			}
		}

		if !hasConnected {
			for _, peer := range room.Members {
				peer.Connected = false
			}
			delete(s.rooms, roomID)
			s.logger.Info("Cleaned up empty room %s", roomID)
		}
		room.mu.Unlock()

		if room.CreatedAt.Before(now.Add(-24 * time.Hour)) && !hasConnected {
			delete(s.rooms, roomID)
			s.logger.Info("Cleaned up stale room %s", roomID)
		}
	}
}

func (s *Server) RoomMemberCount(roomID string) int {
	s.mu.RLock()
	room, ok := s.rooms[roomID]
	s.mu.RUnlock()

	if !ok {
		return 0
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	count := 0
	for _, peer := range room.Members {
		if peer.Connected {
			count++
		}
	}
	return count
}
