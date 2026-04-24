package api

import (
	"context"
	"encoding/json"
	"finanzas-personales/api/db/modelos"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Totales struct {
	TotalCredito    float64 `bson:"totalCredito"`
	TotalDisponible float64 `bson:"totalDisponible"`
	TotalAhorro     float64 `bson:"totalAhorro"`
	TotalApalancado float64 `bson:"totalApalancado"`
	TotalMsi        float64 `bson:"totalMsi"`
}

func TotalsHandler(mongoClient *mongo.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		datos, err := ObtenerTotales(mongoClient)
		if err != nil {
			log.Printf("Error obteniendo totales: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(datos)
	}
}

// ObtenerTotales hace la suma directamente en MongoDB
func ObtenerTotales(mongoClient *mongo.Client) (modelos.Totales, error) {
	var totales modelos.Totales

	collection := mongoClient.Database(dbName).Collection(collectionName) // Usa tus constantes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "totalCredito", Value: bson.D{{Key: "$sum", Value: "$credito"}}},
			{Key: "totalDisponible", Value: bson.D{{Key: "$sum", Value: "$disponible"}}},
			{Key: "totalAhorro", Value: bson.D{{Key: "$sum", Value: "$tener"}}},
			{Key: "totalApalancado", Value: bson.D{{Key: "$sum", Value: "$apalancamiento"}}},
			{Key: "totalMsi", Value: bson.D{{Key: "$sum", Value: "$msi"}}},
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return totales, err
	}
	defer cursor.Close(ctx)

	resultadoAgregacion := Totales{}

	// 4. Si hay resultados, los decodificamos
	if cursor.Next(ctx) {
		if err := cursor.Decode(&resultadoAgregacion); err != nil {
			return totales, err
		}
	} else {
		// Si no hay tarjetas, retornamos los totales en 0
		return totales, nil
	}

	// 5. Pasamos los datos de la base de datos a tu estructura Totales original
	totales.TotalCredito = resultadoAgregacion.TotalCredito
	totales.TotalDisponible = resultadoAgregacion.TotalDisponible
	totales.TotalAhorro = resultadoAgregacion.TotalAhorro
	totales.TotalApalancado = resultadoAgregacion.TotalApalancado
	totales.TotalMsi = resultadoAgregacion.TotalMsi

	// 6. Calculamos los campos derivados en Go (es más fácil hacerlo aquí que en Mongo)
	totales.TotalUsado = totales.TotalCredito - totales.TotalDisponible
	if totales.TotalCredito > 0 {
		totales.UtilizacionGlobal = (totales.TotalUsado / totales.TotalCredito) * 100
	}

	return totales, nil
}
