package main

import "sync"

type EstateRoleStore interface {
	GetRole(userID, estateID string) (string, bool)
	SetRole(userID, estateID, role string)
}

type memoryEstateRoleStore struct {
	mu    sync.RWMutex
	roles map[string]map[string]string
}

func newMemoryEstateRoleStore() *memoryEstateRoleStore {
	store := &memoryEstateRoleStore{roles: make(map[string]map[string]string)}
	store.SetRole("user-1", "estate-1", "admin")
	return store
}

func (s *memoryEstateRoleStore) GetRole(userID, estateID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	estates, ok := s.roles[userID]
	if !ok {
		return "", false
	}

	role, ok := estates[estateID]
	return role, ok
}

func (s *memoryEstateRoleStore) SetRole(userID, estateID, role string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.roles[userID]; !ok {
		s.roles[userID] = make(map[string]string)
	}
	s.roles[userID][estateID] = role
}
