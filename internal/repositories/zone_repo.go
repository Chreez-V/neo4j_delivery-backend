package repositories

import (
	"context"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"neo4j_delivery/internal/models"
)

type ZoneRepository struct {
	Driver neo4j.Driver
}

func NewZoneRepository(driver neo4j.Driver) *ZoneRepository {
	return &ZoneRepository{Driver: driver}
}

func (r *ZoneRepository) FindAll(ctx context.Context) ([]models.Zone, error) {
	session := r.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query := `
		MATCH (z:Zona)
		RETURN z.nombre AS nombre, z.tipo_zona AS tipo
		ORDER BY z.nombre
		`
		result, err := tx.Run(query, nil)
		if err != nil {
			return nil, err
		}

		var zones []models.Zone
		for result.Next() {
			record := result.Record()
			zones = append(zones, models.Zone{
				Nombre:   record.Values[0].(string),
				TipoZona: record.Values[1].(string),
			})
		}

		return zones, nil
	})

	if err != nil {
		return nil, fmt.Errorf("error fetching zones: %w", err)
	}

	return result.([]models.Zone), nil
}

func (r *ZoneRepository) FindOptimalRoute(ctx context.Context, from, to string) ([]models.Connection, error) {
	session := r.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query := `
		MATCH (start:Zona {nombre: $from}), (end:Zona {nombre: $to})
		CALL apoc.algo.dijkstra(start, end, 'CONECTA', 'tiempo_minutos') 
		YIELD path, weight
		UNWIND relationships(path) AS rel
		RETURN 
			startNode(rel).nombre AS source,
			endNode(rel).nombre AS target,
			rel.tiempo_minutos AS tiempo,
			rel.trafico_actual AS trafico,
			rel.capacidad AS capacidad
		`
		params := map[string]interface{}{"from": from, "to": to}
		result, err := tx.Run(query, params)
		if err != nil {
			return nil, err
		}

		var connections []models.Connection
		for result.Next() {
			record := result.Record()
			connections = append(connections, models.Connection{
				Source:    record.Values[0].(string),
				Target:    record.Values[1].(string),
				Tiempo:    int(record.Values[2].(int64)),
				Trafico:   record.Values[3].(string),
				Capacidad: int(record.Values[4].(int64)),
				Direccion: "uni",
			})
		}

		return connections, nil
	})

	if err != nil {
		return nil, fmt.Errorf("error finding optimal route: %w", err)
	}

	return result.([]models.Connection), nil
}
