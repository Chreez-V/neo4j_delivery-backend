package dijkstra

import (
	"container/heap" // Importar el paquete heap
	"fmt"
	"math"
	"neo4j_delivery/internal/models"
)

// Item representa un nodo en la cola de prioridad.
type Item struct {
	Value    string
	Priority float64
	Index    int // Índice de la cola de prioridad.
}

// PriorityQueue implementa la interfaz heap.Interface.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Priority < pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*Item)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // evitar fugas de memoria
	item.Index = -1 // por seguridad
	*pq = old[0 : n-1]
	return item
}

// GetNodes obtiene todos los nodos del grafo.
func GetNodes(graph models.Graph) []string {
	var nodes []string
	seen := make(map[string]bool)

	for key := range graph {
		if !seen[key] {
			nodes = append(nodes, key)
			seen[key] = true
		}
		for _, neighbor := range graph[key] {
			if !seen[neighbor.Item] {
				nodes = append(nodes, neighbor.Item)
				seen[neighbor.Item] = true
			}
		}
	}
	return nodes
}

// Dijkstra implementa el algoritmo de Dijkstra para encontrar los caminos más cortos desde un nodo inicial.
// Ahora devuelve tanto las distancias como los predecesores.
func Dijkstra(graph models.Graph, start string) (map[string]float64, map[string]string) {
	distances := make(map[string]float64)
	previous := make(map[string]string) // Mapa para almacenar los predecesores
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	allNodes := GetNodes(graph) // Obtener todos los nodos para inicializarlos

	for _, node := range allNodes {
		distances[node] = math.Inf(1) // Infinito
		previous[node] = ""
	}
	distances[start] = 0

	heap.Push(&pq, &Item{Value: start, Priority: 0})

	for pq.Len() > 0 {
		current := heap.Pop(&pq).(*Item)
		u := current.Value

		// Si ya encontramos un camino más corto a `u`, ignoramos este.
		if current.Priority > distances[u] {
			continue
		}

		// Si el nodo actual no tiene aristas, continuamos
		if _, ok := graph[u]; !ok {
			continue
		}

		for _, edge := range graph[u] {
			// **¡Aquí la clave! Solo considera aristas accesibles**
			if !edge.Accesible {
				continue
			}

			v := edge.Item
			alt := distances[u] + edge.Cost
			if alt < distances[v] {
				distances[v] = alt
				previous[v] = u // Almacenar el predecesor
				heap.Push(&pq, &Item{Value: v, Priority: alt})
			}
		}
	}
	return distances, previous
}

// Travel reconstruye el camino y el costo desde la tabla de distancias y predecesores.
// Ahora acepta el mapa `previous`.
func Travel(distances map[string]float64, previous map[string]string, start, end string) ([]string, float64, error) {
	path := []string{}
	cost, ok := distances[end]

	if !ok {
		return nil, 0.0, fmt.Errorf("Travel error: destination node '%s' not found in Dijkstra's table (might not exist in graph)", end)
	}
	if cost == math.Inf(1) {
		return nil, 0.0, fmt.Errorf("Travel error: node '%s' is not reachable from '%s'", end, start)
	}

	currentNode := end
	for currentNode != "" && currentNode != start {
		path = append([]string{currentNode}, path...)
		predecessor, exists := previous[currentNode]
		if !exists { // Si no hay predecesor y no es el inicio, el camino no se puede reconstruir
			return nil, 0.0, fmt.Errorf("Travel error: could not reconstruct path to '%s' (predecessor missing for %s)", end, currentNode)
		}
		currentNode = predecessor
	}

	if currentNode == start {
		path = append([]string{start}, path...)
	} else {
		return nil, 0.0, fmt.Errorf("Travel error: could not reconstruct path completely from '%s' to '%s'", start, end)
	}

	return path, cost, nil
}

// FindInaccessibleNodes encuentra nodos inaccesibles desde un nodo de inicio, considerando solo aristas accesibles.
func FindInaccessibleNodes(graph models.Graph, startNode string) ([]string, []string) {
	if len(graph) == 0 {
		return []string{}, []string{}
	}

	visited := make(map[string]bool)
	queue := []string{}

	_, startNodeExists := graph[startNode]
	if !startNodeExists {
		inaccessible := []string{}
		for node := range graph {
			inaccessible = append(inaccessible, node)
		}
		return []string{}, inaccessible
	}

	queue = append(queue, startNode)
	visited[startNode] = true

	for len(queue) > 0 {
		currentNode := queue[0]
		queue = queue[1:]

		// Si el nodo actual no tiene aristas, continuamos
		if _, ok := graph[currentNode]; !ok {
			continue
		}

		for _, neighbor := range graph[currentNode] {
			// Solo considera las aristas marcadas como accesibles
			if neighbor.Accesible && !visited[neighbor.Item] {
				visited[neighbor.Item] = true
				queue = append(queue, neighbor.Item)
			}
		}
	}

	var accessibleNodes []string
	var inaccessibleNodes []string

	for _, node := range GetNodes(graph) { // Iterar sobre todos los nodos conocidos del grafo
		if visited[node] {
			accessibleNodes = append(accessibleNodes, node)
		} else {
			inaccessibleNodes = append(inaccessibleNodes, node)
		}
	}

	return accessibleNodes, inaccessibleNodes
}
