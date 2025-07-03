package repositories

import (
	"context"
	"fmt"
	"neo4j_delivery/internal/models"
	"strconv"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type ZoneRepository struct {
	Driver neo4j.Driver
}

func NewZoneRepository(driver neo4j.Driver) *ZoneRepository {
	return &ZoneRepository{Driver: driver}
}

func (r *ZoneRepository) GetGraphData() (models.GraphData, error) {
	session := r.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query := `
		MATCH (n)
		OPTIONAL MATCH (n)-[r:CONECTA]->(m)
		RETURN n, r, m
		`
		result, err := tx.Run(query, nil)
		if err != nil {
			return nil, err
		}

		nodes := make(map[string]models.Node)
		links := []models.Link{}

		for result.Next() {
			record := result.Record()

			// Procesar nodo origen
			if nodeVal, ok := record.Get("n"); ok && nodeVal != nil {
				if node, ok := nodeVal.(neo4j.Node); ok {
					nodeId := strconv.FormatInt(node.Id, 10)

					// Obtener propiedades con comprobación de nil
					nombre := ""
					if val, ok := node.Props["nombre"].(string); ok {
						nombre = val
					}

					tipoZona := ""
					if val, ok := node.Props["tipo_zona"].(string); ok {
						tipoZona = val
					}

					// Determinar si es centro de distribución
					isCentro := false
					for _, label := range node.Labels {
						if label == "CentroDistribucion" {
							isCentro = true
							break
						}
					}

					label := "Zona"
					if isCentro {
						label = "CentroDistribucion"
					}

					nodes[nodeId] = models.Node{
						ID:    nodeId,
						Name:  nombre,
						Label: label,
						Tipo:  tipoZona,
					}
				}
			}

			// Procesar relación si existe
			if relVal, ok := record.Get("r"); ok && relVal != nil {
				if rel, ok := relVal.(neo4j.Relationship); ok {
					if targetVal, ok := record.Get("m"); ok && targetVal != nil {
						if targetNode, ok := targetVal.(neo4j.Node); ok {
							targetId := strconv.FormatInt(targetNode.Id, 10)

							// Obtener propiedades con valores por defecto
							tiempo := 0.0
							if val, ok := rel.Props["tiempo_minutos"].(int64); ok {
								tiempo = float64(val)
							} else if val, ok := rel.Props["tiempo_minutos"].(float64); ok {
								tiempo = val
							}

							trafico := ""
							if val, ok := rel.Props["trafico_actual"].(string); ok {
								trafico = val
							}

							capacidad := 0
							if val, ok := rel.Props["capacidad"].(int64); ok {
								capacidad = int(val)
							}

							accesible := true
							if val, ok := rel.Props["accesible"].(bool); ok {
								accesible = val
							}

							links = append(links, models.Link{
								Source:         strconv.FormatInt(rel.StartId, 10),
								Target:         targetId,
								Tiempo_minutos: tiempo,
								Trafico_actual: trafico,
								Capacidad:      capacidad,
								Accesible:      accesible,
							})
						}
					}
				}
			}
		}

		// Convertir el mapa de nodos a slice
		nodeSlice := make([]models.Node, 0, len(nodes))
		for _, node := range nodes {
			nodeSlice = append(nodeSlice, node)
		}

		return models.GraphData{
			Nodes: nodeSlice,
			Links: links,
		}, nil
	})

	if err != nil {
		return models.GraphData{}, err
	}

	return result.(models.GraphData), nil
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

func getNodeLabel(node neo4j.Node) string {
	labels := node.Labels
	for _, label := range labels {
		if label == "CentroDistribucion" {
			return "CentroDistribucion"
		}
	}
	return "Zona"
}

func (r *ZoneRepository) GetAllAsGraph() (models.Graph, error) {

	query := `MATCH (n) 
	OPTIONAL MATCH (n)-[z:CONECTA]->(neighbor)
	RETURN n.nombre AS padre,
	z.tiempo_minutos AS tiempo, 
	z.accesible AS accesible,
	neighbor.nombre AS hijo`

	session := r.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	t, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(query, nil)
		if err != nil {
			return nil, err
		}
		g := make(models.Graph)

		for result.Next() {
			var n any
			record := result.Record()
			data := record.AsMap()
			parent := data["padre"].(string)
			if data["hijo"] == nil {
				g[parent] = []models.Edge{}
			} else {
				parsedTime := float64(data["tiempo"].(int64))
				accesible := data["accesible"].(bool)
				n = models.Edge{data["hijo"].(string), accesible, parsedTime}
				g[parent] = append(g[parent], n.(models.Edge))
			}
		}
		return g, nil
	})
	if err != nil {
		return nil, err
	}

	return t.(models.Graph), nil
}

func hasNode(m *models.Graph, target string) bool {
	for key := range *m {
		if key == target {
			return true
		}
	}
	return false
}

// UpdateConnectionAccessibility actualiza la propiedad 'accesible' de una conexión.
// Esta función opera en una dirección (source -> target). Si la relación es bidireccional,
// deberás llamar a esta función para ambas direcciones si quieres cerrar/abrir completamente la "calle".
func (r *ZoneRepository) UpdateConnectionAccessibility(ctx context.Context, source, target string, accessible bool) error {
	session := r.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query := `
		MATCH (s:Zona {nombre: $source})-[r:CONECTA]->(t:Zona {nombre: $target})
		SET r.accesible = $accessible
		RETURN r
		`
		params := map[string]interface{}{
			"source":     source,
			"target":     target,
			"accessible": accessible,
		}
		_, err := tx.Run(query, params)
		if err != nil {
			return nil, fmt.Errorf("failed to update accessibility for %s -> %s: %w", source, target, err)
		}
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("error updating connection accessibility: %w", err)
	}
	return nil
}

// GetConnectionStatus obtiene el estado de accesibilidad de una conexión.
func (r *ZoneRepository) GetConnectionStatus(ctx context.Context, source, target string) (bool, error) {
	session := r.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query := `
		MATCH (s:Zona {nombre: $source})-[r:CONECTA]->(t:Zona {nombre: $target})
		RETURN r.accesible AS accesible
		`
		params := map[string]interface{}{
			"source": source,
			"target": target,
		}
		res, err := tx.Run(query, params)
		if err != nil {
			return false, err
		}
		if res.Next() {
			val, ok := res.Record().Get("accesible")
			if !ok {
				return false, fmt.Errorf("property 'accesible' not found for connection %s -> %s", source, target)
			}
			return val.(bool), nil
		}
		return false, fmt.Errorf("connection not found between %s and %s", source, target)
	})

	if err != nil {
		return false, fmt.Errorf("error getting connection status: %w", err)
	}
	return result.(bool), nil
}

// UpdateConnectionTime actualiza la propiedad 'tiempo_minutos' de una conexión.
func (r *ZoneRepository) UpdateConnectionTime(ctx context.Context, source, target string, newTime float64) error {
	session := r.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query := `
		MATCH (s:Zona {nombre: $source})-[r:CONECTA]->(t:Zona {nombre: $target})
		SET r.tiempo_minutos = $newTime
		RETURN r
		`
		params := map[string]interface{}{
			"source":  source,
			"target":  target,
			"newTime": newTime,
		}
		_, err := tx.Run(query, params)
		if err != nil {
			return nil, fmt.Errorf("falló la actualización de tiempo para %s -> %s: %w", source, target, err)
		}
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("error al actualizar el tiempo de conexión: %w", err)
	}
	return nil
}
