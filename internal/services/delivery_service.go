package services

import (
	"context"
	"neo4j_delivery/internal/models"
	"neo4j_delivery/internal/repositories"
)

type DeliveryService struct {
	zoneRepo *repositories.ZoneRepository
}

func NewDeliveryService(zoneRepo *repositories.ZoneRepository) *DeliveryService {
	return &DeliveryService{zoneRepo: zoneRepo}
}

func (s *DeliveryService) GetAllZones(ctx context.Context) ([]models.Zone, error) {
	return s.zoneRepo.FindAll(ctx)
}

func (s *DeliveryService) CalculateRoute(ctx context.Context, from, to string) ([]models.Connection, error) {
	return s.zoneRepo.FindOptimalRoute(ctx, from, to)
}
