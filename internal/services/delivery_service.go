// services/delivery_services.go
package services

import (
	"context"
	"fmt"
	"neo4j_delivery/internal/models"
	"neo4j_delivery/internal/repositories"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j" // Ensure this import is present
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

func (s *DeliveryService) CloseConnection(ctx context.Context, from, to string) error {
	return s.zoneRepo.CloseConnection(ctx, from, to)
}

func (s *DeliveryService) OpenConnection(ctx context.Context, from, to string) error {
	return s.zoneRepo.OpenConnection(ctx, from, to)
}

// CreateZoneWithConnections añade una nueva zona o centro de distribución con sus conexiones
func (s *DeliveryService) CreateZoneWithConnections(ctx context.Context, zoneData interface{}, connections []models.Connection) error {
	var newZone models.Zone
	isCD := false
	var cdCapacity int

	// Determinar el tipo de zona para crear el nodo correctamente
	if z, ok := zoneData.(models.Zone); ok {
		newZone = z
	} else if cd, ok := zoneData.(models.DistributionCenter); ok {
		newZone = cd.Zone 
		cdCapacity = cd.CapacidadVehiculos
		isCD = true
	} else {
		return fmt.Errorf("tipo de zona inválido proporcionado")
	}


	session := s.zoneRepo.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	
	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		// Crear el nodo de la nueva zona
		createZoneQuery := `
        CREATE (z:Zona {nombre: $nombre, tipo_zona: $tipoZona})
        `
		zoneParams := map[string]interface{}{
			"nombre":   newZone.Nombre,
			"tipoZona": newZone.TipoZona,
		}

		if isCD {
			createZoneQuery = `
            CREATE (z:CentroDistribucion:Zona {nombre: $nombre, tipo_zona: $tipoZona, capacidad_vehiculos: $capacidadVehiculos})
            `
			zoneParams["capacidadVehiculos"] = cdCapacity
		} else if newZone.Poblacion != nil {
			zoneParams["poblacion"] = *newZone.Poblacion
		}

		_, err := tx.Run(createZoneQuery, zoneParams)
		if err != nil {
			return nil, fmt.Errorf("error creating zone %s: %w", newZone.Nombre, err)
		}

		// Crear las relaciones
		for _, conn := range connections {
			createConnQuery := `
            MATCH (from:Zona {nombre: $from}), (to:Zona {nombre: $to})
            CREATE (from)-[:CONECTA {tiempo_minutos: $tiempo, trafico_actual: $trafico, capacidad: $capacidad, activa: TRUE}]->(to)
            `
			connParams := map[string]interface{}{
				"from":      conn.Source,
				"to":        conn.Target,
				"tiempo":    conn.Tiempo,
				"trafico":   conn.Trafico,
				"capacidad": conn.Capacidad,
			}

			_, err := tx.Run(createConnQuery, connParams)
			if err != nil {
				return nil, fmt.Errorf("error creating connection from %s to %s: %w", conn.Source, conn.Target, err)
			}
		}
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("error in CreateZoneWithConnections: %w", err)
	}
	return nil
}

// UpdateConnectionTime actualiza el tiempo_minutos de una conexión
func (s *DeliveryService) UpdateConnectionTime(ctx context.Context, from, to string, newTime int) error {
	return s.zoneRepo.UpdateConnectionTime(ctx, from, to, newTime)
}
