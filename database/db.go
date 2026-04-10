package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func Connect() {
	fmt.Println("Connect() called")
	var err error
	DB, err = sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/nooboj")
	if err != nil {
		panic("SQL Open failed: " + err.Error())
	}

	err = DB.Ping()
	if err != nil {
		panic("Ping failed: " + err.Error())
	}

	fmt.Println("Database connected")
	runSchema()
	log.Println("Database connected and schema applied.")
}

func runSchema() {
	schemaBytes, err := os.ReadFile("database/schema.sql")
	if err != nil {
		log.Fatal("Error reading schema.sql: ", err)
	}

	schema := string(schemaBytes)
	statements := strings.Split(schema, ";")

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		_, err := DB.Exec(stmt)
		if err != nil {
			log.Fatalf("Error executing statement:\n%s\nError: %s", stmt, err)
		}
	}
}
