package services

import (
	"context"
	"log"
	"neo4j_delivery/dijkstra.go"
	"neo4j_delivery/internal/models"
	"neo4j_delivery/internal/repositories"
)

type DeliveryService struct {
	ZoneRepo  *repositories.ZoneRepository
	RouteRepo *repositories.RouteRepository
}

func (s *DeliveryService) GetGraphData() (models.GraphData, error) {
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

// FindShortestPath ahora usa la versión de Dijkstra que devuelve el mapa de predecesores.
func (s *DeliveryService) FindShortestPath(start string, end string) ([]string, float64, error) {
	g, err := s.ZoneRepo.GetAllAsGraph()
	if err != nil {
		return nil, -1, err
	}

	// Llama a la versión actualizada de Dijkstra
	distances, previous := dijkstra.Dijkstra(g, start)

	path, cost, err := dijkstra.Travel(distances, previous, start, end)
	if err != nil {
		return nil, -1, err
	}

	return path, cost, nil
}

func (s *DeliveryService) FindInaccesible(start string) ([]string, []string) {
	g, err := s.ZoneRepo.GetAllAsGraph()
	if err != nil {
		log.Printf("Error getting graph for inaccessible nodes: %v", err)
		return nil, nil
	}
	accesibleNodes, innaccesibleNodes := dijkstra.FindInaccessibleNodes(g, start)
	return accesibleNodes, innaccesibleNodes
}

func (s *DeliveryService) FindDirectAccessible(start string, minutes float64) map[string][]models.Route {
	g, err := s.ZoneRepo.GetAllAsGraph()
	if err != nil {
		log.Printf("Error getting graph for direct accessible nodes: %v", err)
		return nil
	}
	// Usamos FindInaccessibleNodes para obtener los nodos accesibles después de considerar las aristas.
	accesibleNodes, _ := dijkstra.FindInaccessibleNodes(g, start)

	// Ahora usamos la nueva versión de Dijkstra que considera 'accesible' y devuelve 'previous'.
	distances, previous := dijkstra.Dijkstra(g, start)

	result := make(map[string][]models.Route)

	for _, node := range accesibleNodes {
		if node == start { // No queremos la ruta del nodo a sí mismo
			continue
		}
		path, travelTime, err := dijkstra.Travel(distances, previous, start, node)
		if err == nil && travelTime < minutes {
			if _, ok := result[start]; !ok {
				result[start] = []models.Route{}
			}
			result[start] = append(result[start], models.Route{Path: path, Time: travelTime, Target: node})
		} else if err != nil {
			log.Printf("Error calculating path from %s to %s for direct accessible: %v", start, node, err)
		}
	}
	return result
}

func (s *DeliveryService) GetHighTrafficRoutes() []models.Connection {
	routes := s.RouteRepo.GetHighTrafficEdges()
	log.Println(routes)
	return routes
}

// CloseStreet simula el cierre de una calle (conexión) entre dos zonas.
func (s *DeliveryService) CloseStreet(ctx context.Context, source, target string) error {
	return s.ZoneRepo.UpdateConnectionAccessibility(ctx, source, target, false)
}

// OpenStreet reabre una calle (conexión) entre dos zonas.
func (s *DeliveryService) OpenStreet(ctx context.Context, source, target string) error {
	return s.ZoneRepo.UpdateConnectionAccessibility(ctx, source, target, true)
}

// GetStreetStatus obtiene el estado de accesibilidad de una calle.
func (s *DeliveryService) GetStreetStatus(ctx context.Context, source, target string) (bool, error) {
	return s.ZoneRepo.GetConnectionStatus(ctx, source, target)
}
