package main

import (
	"context"
	api "finanzas-personales/api"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	Port = ":8000"
)

var mongoClient *mongo.Client

func main() {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017/finanzas"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Error conectando a MongoDB:", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("No se pudo hacer ping a MongoDB:", err)
	}

	log.Println("Conectado exitosamente a MongoDB")
	mongoClient = client
	api.InyectarDatosDesdeCSV(mongoClient)

	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/api/tarjetas", api.TarjetasHandler(mongoClient))
	http.HandleFunc("/api/dashboard/totals", api.TotalsHandler(mongoClient))

	fmt.Printf("Servidor Go corriendo en http://localhost%s\n", Port)
	log.Fatal(http.ListenAndServe(Port, nil))
}
