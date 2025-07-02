package repositories

import (
	"fmt"
	"neo4j_delivery/internal/models"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// RouteRepository handles database operations related to routes.
type RouteRepository struct {
	Driver neo4j.Driver
}

// NewRouteRepository creates a new instance of RouteRepository.
func NewRouteRepository(driver neo4j.Driver) *RouteRepository {
	return &RouteRepository{Driver: driver}
}

// GetHighTrafficEdges retrieves connections (routes) with high traffic.
func (r *RouteRepository) GetHighTrafficEdges() []models.Connection {
	session := r.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	var connections []models.Connection

	_, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query := `
		MATCH (s:Zona)-[r:CONECTA]->(t:Zona)
		WHERE r.trafico_actual = 'alto'
		RETURN s.nombre AS source, t.nombre AS target, r.tiempo_minutos AS tiempo, r.trafico_actual AS trafico, r.capacidad AS capacidad, r.direccion AS direccion
		`
		result, err := tx.Run(query, nil)
		if err != nil {
			return nil, fmt.Errorf("error running query for high traffic edges: %w", err)
		}

		for result.Next() {
			record := result.Record()
			source := record.Values[0].(string)
			target := record.Values[1].(string)
			tiempo := int(record.Values[2].(int64))
			trafico := record.Values[3].(string)
			capacidad := int(record.Values[4].(int64))
			// Check if 'direccion' exists and is a string, otherwise default to "uni"
			direccion := "uni" // default value
			if val, ok := record.Values[5].(string); ok {
				direccion = val
			}

			connections = append(connections, models.Connection{
				Source:    source,
				Target:    target,
				Tiempo:    tiempo,
				Trafico:   trafico,
				Capacidad: capacidad,
				Direccion: direccion,
			})
		}
		return nil, nil
	})

	if err != nil {
		fmt.Printf("Error in GetHighTrafficEdges: %v\n", err)
		return nil
	}

	return connections
}
