// models/zone.go
package models

type Zone struct {
    ID          string `json:"id"`
    Nombre      string `json:"nombre"`
    TipoZona    string `json:"tipo_zona"`
    Poblacion   *int   `json:"poblacion,omitempty"`
}

type DistributionCenter struct {
    Zone
    CapacidadVehiculos int `json:"capacidad_vehiculos"`
}

type Connection struct {
    Source      string `json:"source"`
    Target      string `json:"target"`
    Tiempo      int    `json:"tiempo_minutos"`
    Trafico     string `json:"trafico_actual"`
    Capacidad   int    `json:"capacidad"`
    Direccion   string `json:"direccion"` // 'uni' o 'bi'
    Activa      bool   `json:"activa"`    //  campo para el estado de la conexión
}
