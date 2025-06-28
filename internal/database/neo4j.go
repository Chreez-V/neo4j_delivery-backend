package database

import (
	"fmt"
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

// ExecuteCypherFile ejecuta un archivo .cypher completo
func (db *Neo4jDatabase) ExecuteCypherFile(filePath string) error {
	session := db.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Leer el archivo .cypher
	cypher, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading cypher file: %w", err)
	}

	_, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		_, err := tx.Run(string(cypher), nil)
		return nil, err
	})

	return err
}
