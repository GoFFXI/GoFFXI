package router

import (
	"fmt"
	"net"
	"time"

	"github.com/GoFFXI/GoFFXI/internal/database"
)

type Session struct {
	clientAddr *net.UDPAddr
	character  *database.Character
	lastUpdate time.Time

	lastClientPacketID uint16
	lastServerPacketID uint16

	sessionKey       string
	currentBlowfish  *Blowfish
	previousBlowfish *Blowfish
}

func NewSession(clientAddr *net.UDPAddr, sessionKey string) (*Session, error) {
	blowfish, err := NewBlowfish(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create blowfish for session: %w", err)
	}

	session := &Session{
		clientAddr:      clientAddr,
		character:       nil,
		lastUpdate:      time.Now(),
		sessionKey:      sessionKey,
		currentBlowfish: blowfish,
	}

	return session, nil
}

func (s *Session) IncrementBlowfish() error {
	// Save the current key as previous
	s.previousBlowfish = s.currentBlowfish

	// Create new Blowfish with incremented key
	newBF := &Blowfish{
		key:    s.currentBlowfish.key,
		status: BlowfishPendingZone,
	}

	// Increment the key
	if err := newBF.IncrementKey(); err != nil {
		return err
	}

	s.currentBlowfish = newBF
	return nil
}
