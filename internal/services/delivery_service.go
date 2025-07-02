package services

import (
	"context"
	"neo4j_delivery/internal/dijkstra"
	"neo4j_delivery/internal/models"
	"neo4j_delivery/internal/repositories"
)

type DeliveryService struct {
	ZoneRepo *repositories.ZoneRepository
}

func (s *DeliveryService) GetGraphData() (models.GraphData, error) {
	// Implementa la lógica para obtener nodos y relaciones de Neo4j
	return s.ZoneRepo.GetGraphData()
}

func NewDeliveryService(ZoneRepo *repositories.ZoneRepository) *DeliveryService {
	return &DeliveryService{ZoneRepo: ZoneRepo}
}

func (s *DeliveryService) GetAllZones(ctx context.Context) ([]models.Zone, error) {
	return s.ZoneRepo.FindAll(ctx)
}

func (s *DeliveryService) CalculateRoute(ctx context.Context, from, to string) ([]models.Connection, error) {
	return s.ZoneRepo.FindOptimalRoute(ctx, from, to)
}
func (s *DeliveryService) FindShortestPath(start string, end string) ([]string, float64, error) {
	g, err := s.ZoneRepo.GetAllAsGraph()
	if err != nil {
		return nil, -1, err
	}
	table := dijkstra.Dijkstra(g, start)
	path, cost, err := dijkstra.Travel(table, start, end)
	if err != nil {
		return nil, -1, err
	}

	return path, cost, nil
}

func (s *DeliveryService) FindInaccesible(start string) ([]string, []string) {
	g, err := s.ZoneRepo.GetAllAsGraph()
	if err != nil {
		return nil, nil
	}
	accesibleNodes, innaccesibleNodes := dijkstra.FindInaccessibleNodes(g, start)
	return accesibleNodes, innaccesibleNodes
}

func (s *DeliveryService) FindDirectAccessible(start string, minutes float64) map[string][]models.Route {
	g, err := s.ZoneRepo.GetAllAsGraph()
	if err != nil {
		return nil
	}
	accesibleNodes, _ := dijkstra.FindInaccessibleNodes(g, start)
	table := dijkstra.Dijkstra(g, start)
	result := make(map[string][]models.Route)

	for i := range accesibleNodes {

		path, travelTime, _ := dijkstra.Travel(table, start, accesibleNodes[i])
		if travelTime < minutes && accesibleNodes[i] != start {
			result[start] = append(result[start], models.Route{path, travelTime, accesibleNodes[i]})
		}
	}
	return result
}
