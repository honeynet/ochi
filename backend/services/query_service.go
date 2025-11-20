package services

import (
	"github.com/honeynet/ochi/backend/entities"
	"github.com/honeynet/ochi/backend/repos"
)

type QueryService struct {
	repo *repos.QueryRepo
}

func NewQueryService(repo *repos.QueryRepo) *QueryService {
	return &QueryService{repo: repo}
}

// CreateQuery creates a new query
func (s *QueryService) CreateQuery(ownerID, content, description string, active bool, tags []entities.Tag) (entities.Query, error) {
	return s.repo.Create(ownerID, content, description, active, tags)
}

// GetQueryByID returns a query by ID
func (s *QueryService) GetQueryByID(id string) (entities.Query, error) {
	return s.repo.GetByID(id)
}

// UpdateQuery updates an existing query
func (s *QueryService) UpdateQuery(id, content, description string, active bool, tags []entities.Tag) error {
	return s.repo.Update(id, content, description, active, tags)
}

// DeleteQuery removes a query
func (s *QueryService) DeleteQuery(id string) error {
	return s.repo.Delete(id)
}

// FindQueriesByOwner returns all queries of an owner
func (s *QueryService) FindQueriesByOwner(ownerID string) ([]entities.Query, error) {
	return s.repo.FindByOwnerId(ownerID)
}
