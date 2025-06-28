package database

import (
	"fmt"
	"strings"
	"os"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jDatabase struct {
	Driver neo4j.Driver
}

func NewNeo4jDatabase(uri, username, password string) (*Neo4jDatabase, error) {
	driver, err := neo4j.NewDriver(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("could not create neo4j driver: %w", err)
	}

	// Verificar conexión
	err = driver.VerifyConnectivity()
	if err != nil {
		return nil, fmt.Errorf("failed to verify connection: %w", err)
	}

	return &Neo4jDatabase{Driver: driver}, nil
}

func (db *Neo4jDatabase) Close() error {
	return db.Driver.Close()
}

// ExecuteCypherFile mejorado para manejar transacciones y múltiples sentencias
func (db *Neo4jDatabase) ExecuteCypherFile(filePath string) error {
	session := db.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Leer el archivo .cypher
	cypher, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading cypher file: %w", err)
	}

	// Dividir en sentencias individuales (separadas por ;)
	statements := strings.Split(string(cypher), ";")

	_, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		for _, stmt := range statements {
			// Eliminar espacios en blanco y saltos de línea
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			// Ejecutar cada sentencia
			_, err := tx.Run(stmt, nil)
			if err != nil {
				return nil, fmt.Errorf("error executing statement: %q, error: %w", stmt, err)
			}
		}
		return nil, nil
	})

	return err
}
