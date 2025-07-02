package repositories

import (
	"fmt"
	"neo4j_delivery/internal/models"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type RouteRepository struct {
	Driver neo4j.Driver
}

func NewRouteRepository(driver neo4j.Driver) *RouteRepository {
	return &RouteRepository{Driver: driver}
}

func (r *RouteRepository) GetHighTrafficEdges() []models.Connection {

	query := `MATCH (n)-[z:CONECTA]->(y)
	WHERE z.trafico_actual='alto'
	RETURN ID(z) AS id,
	n.nombre AS source,
	y.nombre AS target,
	z.capacidad AS capacidad,
	z.trafico_actual AS traffic,
	z.tiempo_minutos AS tiempo,
	z.accesible AS accesible`

	session := r.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	t, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(query, nil)
		if err != nil {
			return nil, err
		}
		edges := []models.Connection{}

		for result.Next() {
			record := result.Record()
			data := record.AsMap()
			time := int(data["tiempo"].(int64))
			capacity := int(data["capacidad"].(int64))
			edge := models.Connection{data["source"].(string), data["target"].(string), time, data["traffic"].(string), capacity, "?"}
			edges = append(edges, edge)

		}
		return edges, nil
	})
	if err != nil {
		return nil
	}
	return t.([]models.Connection)
}

// session := r.Driver.NewSession(neo4j.SessionConfig{})
// defer session.Close()
