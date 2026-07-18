package main

import (
	"context"
	"finanzas-personales/api"
	"finanzas-personales/database"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	Port = ":8000"
)

func main() {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.NewDBClient(ctx, uri, "finanzas")
	if err != nil {
		log.Fatalf("Error conectando a la base de datos: %v", err)
	}
	defer db.Close(context.Background())

	log.Println("Conectado exitosamente a MongoDB")

	server := &api.API{
		DB: db,
	}

	http.HandleFunc("/api/tarjetas", server.TarjetasHandler)
	http.HandleFunc("/api/dashboard/totals", server.TotalsHandler)

	fmt.Printf("Servidor Go corriendo en http://localhost%s\n", Port)
	log.Fatal(http.ListenAndServe(Port, nil))
}
