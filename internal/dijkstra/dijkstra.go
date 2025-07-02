package dijkstra

import (
	"fmt"
	"math"
	"neo4j_delivery/internal/models"
)

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

// Inicializa el costo de los nodos que aún no han sido visitados
func InitCosts(graph models.Graph, startingNode string) map[string]models.Edge {
	table := make(map[string]models.Edge)
	allNodes := GetNodes(graph)

	for _, node := range allNodes {
		if node == startingNode {
			table[node] = models.Edge{"", true, 0}
		} else {
			table[node] = models.Edge{"", true, math.Inf(1)}
		}
	}
	return table
}

// Encuentra el nodo no visitado con menor peso en su arista
// Se usa para encontrar el nodo no visitado siguiente a ser procesado
func findMinCostNode(costs map[string]models.Edge, unvisitedNodes map[string]bool) string {
	minCost := math.Inf(1)
	var minNode string
	found := false

	for node, isUnvisited := range unvisitedNodes {
		if isUnvisited {
			if costs[node].Cost < minCost {
				minCost = costs[node].Cost
				minNode = node
				found = true
			}
		}
	}
	if !found {
		return ""
	}
	return minNode
}

func Dijkstra(graph models.Graph, start string) map[string]models.Edge {
	table := InitCosts(graph, start)
	fmt.Println("Initial costs:", table)

	unvisitedNodes := make(map[string]bool)
	for _, node := range GetNodes(graph) {
		unvisitedNodes[node] = true
	}

	for len(unvisitedNodes) > 0 {
		currentNode := findMinCostNode(table, unvisitedNodes)

		if currentNode == "" || table[currentNode].Cost == math.Inf(1) {
			break
		}

		unvisitedNodes[currentNode] = false

		for _, neighbor := range graph[currentNode] {
			newCost := table[currentNode].Cost + neighbor.Cost

			if newCost < table[neighbor.Item].Cost {
				table[neighbor.Item] = models.Edge{currentNode, true, newCost}
			}
		}
		fmt.Println("Current table after processing", currentNode, ":", table)
	}

	fmt.Println("\nFinal costs table:", table)
	return table
}

// Retorna un arreglo de strings con los en orden a recorrer para llegar al destino indicado
func Travel(table map[string]models.Edge, start string, end string) ([]string, float64, error) {
	var path []string
	currentNode := end

	if _, exists := table[end]; !exists {
		return nil, 0.0, fmt.Errorf("Travel error: destination node '%s' not found in Dijkstra's table (might not exist in graph)", end)
	}
	if table[end].Cost == math.Inf(1) {
		return nil, 0.0, fmt.Errorf("Travel error: node '%s' is not reachable from '%s'", end, start)
	}

	for currentNode != "" && currentNode != start {
		path = append([]string{currentNode}, path...)
		predecessor := table[currentNode].Item

		if predecessor == "" && currentNode != start {
			return nil, 0.0, fmt.Errorf("Travel error: could not reconstruct path to '%s' (predecessor missing for %s)", end, currentNode)
		}
		if len(path) > len(table) {
			return nil, 0.0, fmt.Errorf("Travel error: path reconstruction loop detected or invalid table for path from '%s' to '%s'", start, end)
		}
		currentNode = predecessor
	}

	if currentNode == start {
		path = append([]string{start}, path...)
	} else {
		return nil, 0.0, fmt.Errorf("Travel error: could not reconstruct path completely from '%s' to '%s'", start, end)
	}

	totalCost := table[end].Cost

	return path, totalCost, nil
}

func RemoveElementByIndex[T any](slice []T, index int) ([]T, bool) {
	if index < 0 || index >= len(slice) {
		fmt.Printf("Error: Index %d is out of bounds for slice of length %d\n", index, len(slice))
		return slice, false
	}

	return append(slice[:index], slice[index+1:]...), true
}

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

		for _, neighbor := range graph[currentNode] {
			if neighbor.Accesible && !visited[neighbor.Item] {
				visited[neighbor.Item] = true
				queue = append(queue, neighbor.Item)
			}
		}
	}

	var accessibleNodes []string
	var inaccessibleNodes []string

	for node := range graph {
		if visited[node] {
			accessibleNodes = append(accessibleNodes, node)
		} else {
			inaccessibleNodes = append(inaccessibleNodes, node)
		}
	}

	return accessibleNodes, inaccessibleNodes
}
