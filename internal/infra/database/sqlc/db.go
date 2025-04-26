package sqlc

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"go-api/pkg/resource"
	"log"
)

var Db *sql.DB

func init() {
	host := resource.GetString("app.db.host")
	port := resource.GetString("app.db.port")
	password := resource.GetString("app.db.password")
	username := resource.GetString("app.db.username")
	database := resource.GetString("app.db.database")
	schema := resource.GetString("app.db.schema")
	sslMode := "disable"

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s search_path=%s",
		host, port, username, password, database, sslMode, schema)

	var err error
	Db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}

	if err = Db.Ping(); err != nil {
		log.Fatalf("Failed to ping DB: %v", err)
	}
}
