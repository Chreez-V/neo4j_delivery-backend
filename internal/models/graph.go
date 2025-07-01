package models

type Edge struct {
	Item      string
	Accesible bool
	Cost      float64
}

type Graph map[string][]Edge
