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
	"github.com/rs/cors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	service := services.DeliveryService{repositories.NewZoneRepository(db.Driver)}

	defer db.Close()

	// Cargar datos iniciales
	err = db.ExecuteCypherFile("scripts/data.cypher")
	if err != nil {
		log.Printf("Warning: could not initialize DB: %v", err)
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
		log.Println("zones request")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Zonas endpoint funciona"}`))
	})

	router.HandleFunc("/api/route", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		log.Println("route request")
		w.Write([]byte(`{"message": "Rutas endpoint funciona"}`))
	})

	router.HandleFunc("/api/zones/dijkstra", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		queryParams := r.URL.Query()

		start := queryParams.Get("start")
		end := queryParams.Get("end")

		path, cost, err := service.FindShortestPath(start, end)
		if err != nil {
			w.Write([]byte(`error bro`))
		}
		if err != nil {
			log.Fatalf("Error marshalling graph to JSON: %v", err)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"items": path, "minutes": cost})
	})

	router.HandleFunc("/api/zones/accesible", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		queryParams := r.URL.Query()
		start := queryParams.Get("start")
		fmt.Println(start)
		accesible, inaccesible := service.FindInaccesible(start)

		json.NewEncoder(w).Encode(map[string]interface{}{"accesible": accesible, "inaccesible": inaccesible})
	})

	// Configurar CORS
c := cors.New(cors.Options{
    AllowedOrigins:   []string{"http://localhost:5173", "http://127.0.0.1:5173"},
    AllowedMethods:   []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
    AllowCredentials: true,
    Debug:           true, // Solo para desarrollo
})
	// Configurar servidor HTTP con CORS
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: c.Handler(router),
	}

	// Iniciar servidor
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
