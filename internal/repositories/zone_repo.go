// repositories/zone_repositories.go
package repositories

import (
	"context"
	"fmt"
	"neo4j_delivery/internal/models"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
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
        CALL apoc.algo.dijkstra(start, end, 'CONECTA', 'tiempo_minutos') YIELD path, weight
        UNWIND relationships(path) AS rel
        WHERE rel.activa = TRUE OR NOT EXISTS(rel.activa) // Solo considerar relaciones activas o sin propiedad 'activa'
        RETURN
            startNode(rel).nombre AS source,
            endNode(rel).nombre AS target,
            rel.tiempo_minutos AS tiempo,
            rel.trafico_actual AS trafico,
            rel.capacidad AS capacidad,
            COALESCE(rel.activa, TRUE) AS activa // Devuelve TRUE si no existe la propiedad
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
				Direccion: "uni", // La dirección puede necesitar ser inferida o almacenada en la relación
				Activa:    record.Values[5].(bool),
			})
		}

		return connections, nil
	})

	if err != nil {
		return nil, fmt.Errorf("error finding optimal route: %w", err)
	}

	return result.([]models.Connection), nil
}

// CloseConnection simula el cierre de una calle (desactiva la relación)
func (r *ZoneRepository) CloseConnection(ctx context.Context, from, to string) error {
	session := r.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query := `
        MATCH (from:Zona {nombre: $from})-[rel:CONECTA]->(to:Zona {nombre: $to})
        SET rel.activa = FALSE
        RETURN rel
        `
		params := map[string]interface{}{"from": from, "to": to}
		_, err := tx.Run(query, params)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("error closing connection between %s and %s: %w", from, to, err)
	}
	return nil
}

// OpenConnection reabre una calle (activa la relación)
func (r *ZoneRepository) OpenConnection(ctx context.Context, from, to string) error {
	session := r.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query := `
        MATCH (from:Zona {nombre: $from})-[rel:CONECTA]->(to:Zona {nombre: $to})
        SET rel.activa = TRUE
        RETURN rel
        `
		params := map[string]interface{}{"from": from, "to": to}
		_, err := tx.Run(query, params)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("error opening connection between %s and %s: %w", from, to, err)
	}
	return nil
}

// CreateZoneAndConnections añade una nueva Zona/Centro de Distribución y sus relaciones
func (r *ZoneRepository) CreateZoneAndConnections(ctx context.Context, newZoneData interface{}, connections []models.Connection) error {
	session := r.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		var newZone models.Zone
		isCD := false
		var cdCapacity int

		if z, ok := newZoneData.(models.Zone); ok {
			newZone = z
		} else if cd, ok := newZoneData.(models.DistributionCenter); ok {
			newZone = cd.Zone
			cdCapacity = cd.CapacidadVehiculos
			isCD = true
		} else {
			return nil, fmt.Errorf("tipo de zona inválido proporcionado")
		}

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
		return fmt.Errorf("error in CreateZoneAndConnections: %w", err)
	}
	return nil
}

// UpdateConnectionTime actualiza el tiempo_minutos de una conexión entre dos zonas
func (r *ZoneRepository) UpdateConnectionTime(ctx context.Context, from, to string, newTime int) error {
	session := r.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query := `
        MATCH (from:Zona {nombre: $from})-[rel:CONECTA]->(to:Zona {nombre: $to})
        SET rel.tiempo_minutos = $newTime
        RETURN rel
        `
		params := map[string]interface{}{
			"from":    from,
			"to":      to,
			"newTime": newTime,
		}
		_, err := tx.Run(query, params)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("error updating connection time between %s and %s: %w", from, to, err)
	}
	return nil
}
