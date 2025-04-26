package gorm

import (
	"fmt"
	"go-api/pkg/resource"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

var Db *gorm.DB

func init() {
	host := resource.GetString("app.db.host")
	port := resource.GetString("app.db.port")
	password := resource.GetString("app.db.password")
	username := resource.GetString("app.db.username")
	database := resource.GetString("app.db.database")
	schema := resource.GetString("app.db.schema")
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable search_path=%s",
		host, username, password, database, port, schema)

	var err error
	Db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Fail to connect Database", zap.Error(err))
	}
}
