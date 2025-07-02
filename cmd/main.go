// main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"neo4j_delivery/internal/config"
	"neo4j_delivery/internal/database"
	"neo4j_delivery/internal/models"
	"neo4j_delivery/internal/repositories"
	"neo4j_delivery/internal/services"
)

func main() {
	// Configuración
	cfg := config.LoadConfig()

	db, err := database.NewNeo4jDatabase(
		cfg.Neo4jURI,
		cfg.Neo4jUser,
		cfg.Neo4jPassword,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Cargar datos iniciales
	err = db.ExecuteCypherFile("scripts/data.cypher")
	if err != nil {
		log.Printf("Warning: could not initialize DB: %v", err)
	} else {
		log.Println("Datos iniciales cargados correctamente")
	}

	// Inicializar repositorios y servicios
	zoneRepo := repositories.NewZoneRepository(db.Driver)
	deliveryService := services.NewDeliveryService(zoneRepo)

	// Configurar endpoints
	router := http.NewServeMux()

	router.HandleFunc("/api/zones", func(w http.ResponseWriter, r *http.Request) {
		zones, err := deliveryService.GetAllZones(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(zones)
	})

	router.HandleFunc("/api/route", func(w http.ResponseWriter, r *http.Request) {
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		if from == "" || to == "" {
			http.Error(w, "Los parámetros 'from' y 'to' son requeridos.", http.StatusBadRequest)
			return
		}

		route, err := deliveryService.CalculateRoute(r.Context(), from, to)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(route)
	})

	// Nuevo endpoint para cerrar una conexión
	router.HandleFunc("/api/connection/close", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			From string `json:"from"`
			To   string `json:"to"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Solicitud inválida", http.StatusBadRequest)
			return
		}

		if req.From == "" || req.To == "" {
			http.Error(w, "Los parámetros 'from' y 'to' son requeridos.", http.StatusBadRequest)
			return
		}

		err := deliveryService.CloseConnection(r.Context(), req.From, req.To)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error al cerrar conexión: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"message": "Conexión entre %s y %s cerrada exitosamente."}`, req.From, req.To)))
	})

	// Nuevo endpoint para abrir una conexión
	router.HandleFunc("/api/connection/open", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			From string `json:"from"`
			To   string `json:"to"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Solicitud inválida", http.StatusBadRequest)
			return
		}

		if req.From == "" || req.To == "" {
			http.Error(w, "Los parámetros 'from' y 'to' son requeridos.", http.StatusBadRequest)
			return
		}

		err := deliveryService.OpenConnection(r.Context(), req.From, req.To)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error al abrir conexión: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"message": "Conexión entre %s y %s abierta exitosamente."}`, req.From, req.To)))
	})

	// Nuevo endpoint para añadir una nueva zona o centro de distribución
	router.HandleFunc("/api/zone", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		var requestBody struct {
			Zone struct {
				Nombre             string `json:"nombre"`
				TipoZona           string `json:"tipo_zona"`
				Poblacion          *int   `json:"poblacion,omitempty"`
				CapacidadVehiculos *int   `json:"capacidad_vehiculos,omitempty"` // Para centros de distribución
			} `json:"zone"`
			Connections []models.Connection `json:"connections"`
		}

		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, fmt.Sprintf("Solicitud JSON inválida: %v", err), http.StatusBadRequest)
			return
		}

		if requestBody.Zone.Nombre == "" || requestBody.Zone.TipoZona == "" {
			http.Error(w, "Los campos 'nombre' y 'tipo_zona' de la zona son requeridos.", http.StatusBadRequest)
			return
		}

		var newZone interface{}
		if requestBody.Zone.TipoZona == "logistica" {
			if requestBody.Zone.CapacidadVehiculos == nil {
				http.Error(w, "La 'capacidad_vehiculos' es requerida para centros de distribución.", http.StatusBadRequest)
				return
			}
			newZone = models.DistributionCenter{
				Zone: models.Zone{
					Nombre:   requestBody.Zone.Nombre,
					TipoZona: requestBody.Zone.TipoZona,
				},
				CapacidadVehiculos: *requestBody.Zone.CapacidadVehiculos,
			}
		} else {
			newZone = models.Zone{
				Nombre:    requestBody.Zone.Nombre,
				TipoZona:  requestBody.Zone.TipoZona,
				Poblacion: requestBody.Zone.Poblacion,
			}
		}

		// Validar conexiones (opcional, pero recomendado)
		for i := range requestBody.Connections { // Usar range con índice para modificar
			conn := &requestBody.Connections[i] // Obtener un puntero para modificar el elemento
			if conn.Source == "" || conn.Target == "" {
				http.Error(w, "Las conexiones deben especificar 'source' y 'target'.", http.StatusBadRequest)
				return
			}

		}

		err = deliveryService.CreateZoneWithConnections(r.Context(), newZone, requestBody.Connections)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error al crear zona y conexiones: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"message": "Zona '%s' y sus conexiones creadas exitosamente."}`, requestBody.Zone.Nombre)))
	})

	// Nuevo endpoint para actualizar el tiempo de una conexión
	router.HandleFunc("/api/connection/time", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut { // Usar PUT para actualizaciones
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			From    string `json:"from"`
			To      string `json:"to"`
			NewTime int    `json:"new_time"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Solicitud JSON inválida", http.StatusBadRequest)
			return
		}

		if req.From == "" || req.To == "" || req.NewTime <= 0 {
			http.Error(w, "Los parámetros 'from', 'to' y 'new_time' (mayor que 0) son requeridos.", http.StatusBadRequest)
			return
		}

		err := deliveryService.UpdateConnectionTime(r.Context(), req.From, req.To, req.NewTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error al actualizar tiempo de conexión: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"message": "Tiempo de conexión entre %s y %s actualizado a %d minutos."}`, req.From, req.To, req.NewTime)))
	})

	// Iniciar servidor
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	go func() {
		log.Printf("Server starting on port %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start server: %v", err)
		}
	}()

	// Manejar shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
