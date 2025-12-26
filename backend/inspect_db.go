package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	connStr := "postgres://amityadav9314:amit8780@localhost:5432/inkgrid?sslmode=disable"
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tables := []string{"users", "materials", "flashcards", "tags", "material_tags", "daily_articles"}
	for _, table := range tables {
		fmt.Printf("Checking table: %s\n", table)
		var tableName string
		err := db.QueryRow("SELECT table_name FROM information_schema.tables WHERE table_name = $1", table).Scan(&tableName)
		if err != nil {
			log.Fatalf("Table %s not found: %v", table, err)
		}
		fmt.Printf("Table %s exists.\n", tableName)

		rows, err := db.Query(`
			SELECT
				kcu.column_name,
				ccu.table_name AS foreign_table_name,
				ccu.column_name AS foreign_column_name
			FROM
				information_schema.key_column_usage AS kcu
				JOIN information_schema.referential_constraints AS rc
					ON kcu.constraint_name = rc.constraint_name
				JOIN information_schema.constraint_column_usage AS ccu
					ON rc.unique_constraint_name = ccu.constraint_name
			WHERE
				kcu.table_name = $1;
		`, table)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		for rows.Next() {
			var col, fTable, fCol string
			if err := rows.Scan(&col, &fTable, &fCol); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("  - FK: %s -> %s.%s\n", col, fTable, fCol)
		}
	}
}
