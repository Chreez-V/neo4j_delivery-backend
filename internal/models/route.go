package models

type Route struct {
	Path   []string `json:"path"`
	Time   float64  `json:"time"`
	Target string   `json:"target"`
}
