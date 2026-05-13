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

func TotalsHandler(mongoClient *mongo.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var totales struct {
			TotalCredito      float64 `json:"totalCredito"`
			TotalDisponible   float64 `json:"totalDisponible"`
			TotalAhorro       float64 `json:"totalAhorro"`
			TotalApalancado   float64 `json:"totalApalancado"`
			TotalMsi          float64 `json:"totalMsi"`
			TotalUsado        float64 `json:"totalUsado"`
			UtilizacionGlobal float64 `json:"utilizacionGlobal"`
		}

		collection := mongoClient.Database(dbName).Collection(collectionName)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := collection.Find(ctx, bson.D{})
		if err != nil {
			log.Printf("Error obteniendo totales: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}
		defer cursor.Close(ctx)

		var tarjetas []modelos.Tarjeta
		if err = cursor.All(ctx, &tarjetas); err != nil {
			log.Printf("Error obteniendo totales: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}

		for i := range tarjetas {
			tarjetas[i].CalcularCredito()
			totales.TotalCredito += tarjetas[i].Credito
			totales.TotalDisponible += tarjetas[i].Disponible
			totales.TotalAhorro += tarjetas[i].Tener
			totales.TotalApalancado += tarjetas[i].Apalancamiento
			totales.TotalMsi += tarjetas[i].Msi
		}

		totales.TotalUsado = totales.TotalCredito - totales.TotalDisponible
		if totales.TotalCredito > 0 {
			totales.UtilizacionGlobal = (totales.TotalUsado / totales.TotalCredito) * 100
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(totales)
	}
}
