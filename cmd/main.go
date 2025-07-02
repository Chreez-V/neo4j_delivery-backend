package main

import (
	_ "bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"neo4j_delivery/internal/config"
	"neo4j_delivery/internal/database"
	"neo4j_delivery/internal/repositories"
	"neo4j_delivery/internal/services"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/rs/cors"
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
		log.Fatalf("Error al conectar con Neo4j: %v", err)
	}

	service := services.DeliveryService{
		ZoneRepo:  repositories.NewZoneRepository(db.Driver),
		RouteRepo: repositories.NewRouteRepository(db.Driver), // Asegúrate de que RouteRepo esté inicializado si se usa
	}

	defer db.Close()

	// Cargar datos iniciales
	err = db.ExecuteCypherFile("scripts/data.cypher")
	if err != nil {
		log.Printf("Advertencia: No se pudieron cargar los datos iniciales: %v", err)
	} else {
		log.Println("Datos iniciales cargados correctamente")
	}

	// Configurar endpoints
	router := http.NewServeMux()

	router.HandleFunc("/api/graph", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		graphData, err := service.GetGraphData()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(graphData)
	})

	router.HandleFunc("/api/zones", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Solicitud a /api/zones")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Zonas endpoint funciona"}`))
	})

	router.HandleFunc("/api/route", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		log.Println("Solicitud a /api/route")
		w.Write([]byte(`{"message": "Rutas endpoint funciona"}`))
	})

	router.HandleFunc("/api/route/hightraffic", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		route := service.RouteRepo.GetHighTrafficEdges()
		json.NewEncoder(w).Encode(map[string]interface{}{"items": route})
	})

	router.HandleFunc("/api/zones/dijkstra", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		queryParams := r.URL.Query()

		start := queryParams.Get("start")
		end := queryParams.Get("end")

		if start == "" || end == "" {
			http.Error(w, "Parámetros 'start' y 'end' son requeridos.", http.StatusBadRequest)
			return
		}

		path, cost, err := service.FindShortestPath(start, end)
		if err != nil {
			log.Printf("Error al encontrar la ruta más corta de %s a %s: %v", start, end, err)
			http.Error(w, fmt.Sprintf("No se pudo encontrar una ruta: %v", err), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"items": path, "minutes": cost})
	})

	router.HandleFunc("/api/zones/accesible", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		queryParams := r.URL.Query()
		start := queryParams.Get("start")
		direct := queryParams.Get("direct")
		minutesStr := queryParams.Get("minutes")

		if start == "" {
			http.Error(w, "Parámetro 'start' es requerido.", http.StatusBadRequest)
			return
		}

		if direct == "" {
			accesible, inaccesible := service.FindInaccesible(start)
			json.NewEncoder(w).Encode(map[string]interface{}{"accesible": accesible, "inaccesible": inaccesible})
		} else {
			minutes, err := strconv.ParseFloat(minutesStr, 64)
			if err != nil {
				http.Error(w, "Parámetro 'minutes' debe ser un número válido.", http.StatusBadRequest)
				return
			}
			routes := service.FindDirectAccessible(start, minutes)
			json.NewEncoder(w).Encode(map[string]interface{}{"from": start, "to": routes})
		}
	})

	// --- Nuevos Endpoints para Cierre/Reapertura de Calles ---
	router.HandleFunc("/api/street/close", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		var requestBody struct {
			Source string `json:"source"`
			Target string `json:"target"`
		}

		err := json.NewDecoder(r.Body).Decode(&requestBody)
		if err != nil {
			http.Error(w, "Cuerpo de solicitud inválido", http.StatusBadRequest)
			return
		}
		if requestBody.Source == "" || requestBody.Target == "" {
			http.Error(w, "Los parámetros 'source' y 'target' son requeridos en el cuerpo.", http.StatusBadRequest)
			return
		}

		err = service.CloseStreet(r.Context(), requestBody.Source, requestBody.Target)
		if err != nil {
			log.Printf("Error al cerrar calle de %s a %s: %v", requestBody.Source, requestBody.Target, err)
			http.Error(w, fmt.Sprintf("Error al cerrar calle: %v", err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Calle de '%s' a '%s' cerrada exitosamente.", requestBody.Source, requestBody.Target)})
	})

	router.HandleFunc("/api/street/open", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		var requestBody struct {
			Source string `json:"source"`
			Target string `json:"target"`
		}

		err := json.NewDecoder(r.Body).Decode(&requestBody)
		if err != nil {
			http.Error(w, "Cuerpo de solicitud inválido", http.StatusBadRequest)
			return
		}
		if requestBody.Source == "" || requestBody.Target == "" {
			http.Error(w, "Los parámetros 'source' y 'target' son requeridos en el cuerpo.", http.StatusBadRequest)
			return
		}

		err = service.OpenStreet(r.Context(), requestBody.Source, requestBody.Target)
		if err != nil {
			log.Printf("Error al reabrir calle de %s a %s: %v", requestBody.Source, requestBody.Target, err)
			http.Error(w, fmt.Sprintf("Error al reabrir calle: %v", err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Calle de '%s' a '%s' reabierta exitosamente.", requestBody.Source, requestBody.Target)})
	})

	router.HandleFunc("/api/street/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		queryParams := r.URL.Query()
		source := queryParams.Get("source")
		target := queryParams.Get("target")

		if source == "" || target == "" {
			http.Error(w, "Los parámetros de consulta 'source' y 'target' son requeridos", http.StatusBadRequest)
			return
		}

		status, err := service.GetStreetStatus(r.Context(), source, target)
		if err != nil {
			log.Printf("Error al obtener estado de calle de %s a %s: %v", source, target, err)
			http.Error(w, fmt.Sprintf("Error al obtener estado de calle: %v", err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"source":     source,
			"target":     target,
			"accessible": status,
		})
	})
	// --- Fin de Nuevos Endpoints ---

	// Configurar CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
		AllowCredentials: true,
		Debug:            true, // Solo para desarrollo
	})
	// Configurar servidor HTTP con CORS
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: c.Handler(router),
	}

	// Iniciar servidor
	go func() {
		log.Printf("Servidor iniciando en el puerto %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("No se pudo iniciar el servidor: %v", err)
		}
	}()

	// Manejar shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Servidor forzado a apagarse: %v", err)
	}

	log.Println("Servidor saliendo")
}
