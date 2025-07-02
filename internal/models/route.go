package models

type Route struct {
	Path   []string `json:"path"`
	Time   float64  `json:"time_minutes"`
	Target string   `json:"target_node"`
}
