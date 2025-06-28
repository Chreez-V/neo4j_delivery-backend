package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"neo4j_delivery/internal/config"
	"neo4j_delivery/internal/database"
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

	// Configurar endpoints
	router := http.NewServeMux()
router.HandleFunc("/api/zones", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"message": "Zonas endpoint funciona"}`))
})

router.HandleFunc("/api/route", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"message": "Rutas endpoint funciona"}`))
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
