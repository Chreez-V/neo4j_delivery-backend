package models


type GraphData struct {
    Nodes []Node `json:"nodes"`
    Links []Link `json:"links"`
}

type Node struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Label string `json:"label"`
    Tipo  string `json:"tipo,omitempty"`
}

type Link struct {
    Source          string  `json:"source"`
    Target          string  `json:"target"`
    Tiempo_minutos  float64 `json:"tiempo_minutos"`
    Trafico_actual  string  `json:"trafico_actual,omitempty"`
    Capacidad       int     `json:"capacidad,omitempty"`
    Accesible       bool    `json:"accesible,omitempty"`
}

type Edge struct {
	Item      string
	Accesible bool
	Cost      float64
}

type Graph map[string][]Edge
