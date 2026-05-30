package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	dbName = "finanzas"
)

type Tarjeta struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Nombre          string             `bson:"nombre" json:"nombre"`
	Disponible      float64            `bson:"disponible" json:"disponible"`
	Saldo           float64            `bson:"saldo" json:"saldo"`
	Apagar          float64            `bson:"-" json:"apagar"`
	FechaPago       string             `bson:"fechaAPago" json:"fechaPago"`
	Color           string             `bson:"color" json:"color"`
	Credito         float64            `bson:"credito" json:"credito"`
	SaldoAPago      float64            `bson:"saldoAPago" json:"saldoAPago"`
	SemanaAPago     int                `bson:"-" json:"semanaAPago"`
	TenerAPago      float64            `bson:"-" json:"tenerAPago"`
	SemanaCorriente int                `bson:"-" json:"semanaCorriente"`
	TenerCorriente  float64            `bson:"-" json:"tenerCorriente"`
	Tener           float64            `bson:"-" json:"tener"`
	Apalancamiento  float64            `bson:"-" json:"apalancamiento"`
	Msi             float64            `bson:"-" json:"msi"`
	Uso             float64            `bson:"-" json:"uso"`
	UsoPorcentaje   float64            `bson:"-" json:"usoPorcentaje"`
}

func (t *Tarjeta) CalcularCredito() {
	t.Apagar = t.SaldoAPago

	fechaPago, err := time.Parse("2006-01-02", t.FechaPago)
	if err != nil {
		fechaPago, err = time.Parse("02/01/2006", t.FechaPago)
		if err != nil {
			fechaPago = time.Now()
		}
	}

	hoy := time.Now().Truncate(24 * time.Hour)
	diasAlViernes := (int(fechaPago.Weekday()) - 4 + 7) % 7
	inicioSemana7 := fechaPago.AddDate(0, 0, -diasAlViernes)
	diasDiff := int(inicioSemana7.Sub(hoy).Hours() / 24)

	semanas := 7
	if diasDiff > 0 {
		semanas = 7 - int(math.Ceil(float64(diasDiff)/7.0))
	}
	t.SemanaAPago = int(math.Max(1, math.Min(7, float64(semanas))))

	t.TenerAPago = math.Round((t.SaldoAPago*float64(t.SemanaAPago)/7.0)*100) / 100

	saldoCorriente := math.Max(0.0, t.Saldo-t.SaldoAPago)

	t.SemanaCorriente = 1
	if t.SemanaAPago > 4 {
		t.SemanaCorriente = t.SemanaAPago - 4
	}
	t.TenerCorriente = math.Round((saldoCorriente*float64(t.SemanaCorriente)/7.0)*100) / 100

	t.Uso = math.Max(0.0, t.Credito-t.Disponible)
	msiTotal := math.Max(0.0, t.Uso-t.Saldo)
	tenerAcumulado := t.TenerCorriente + t.TenerAPago

	apalancamientoTotal := math.Max(0.0, t.Uso-tenerAcumulado)
	msi := math.Min(msiTotal, apalancamientoTotal)

	t.Tener = math.Min(t.Uso, tenerAcumulado)
	t.Msi = math.Round(msi*100) / 100
	t.Apalancamiento = math.Round(math.Max(0.0, apalancamientoTotal-msi)*100) / 100

	t.UsoPorcentaje = 0.0
	if t.Credito > 0 {
		t.UsoPorcentaje = math.Round((t.Uso/t.Credito*100)*10) / 10
	}

	t.FechaPago = fechaPago.Format("02/01/2006")
}

func TarjetasHandler(mongoClient *mongo.Client) http.HandlerFunc {
	var collectionName = "tarjetas"

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tarjetas" {
			http.NotFound(w, r)
			return
		}

		collection := mongoClient.Database(dbName).Collection(collectionName)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		switch r.Method {

		case http.MethodGet:
			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				findOptions := options.Find()
				findOptions.SetSort(bson.D{{Key: "saldoAPago", Value: -1}})

				cursor, err := collection.Find(ctx, bson.D{}, findOptions)
				if err != nil {
					log.Printf("Error consultando MongoDB: %v", err)
					http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
					return
				}
				defer cursor.Close(ctx)

				var tarjetas []Tarjeta
				if err = cursor.All(ctx, &tarjetas); err != nil {
					log.Printf("Error decodificando resultados: %v", err)
					http.Error(w, "Error interno", http.StatusInternalServerError)
					return
				}

				for i := range tarjetas {
					tarjetas[i].CalcularCredito()
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tarjetas)
				return
			}

			objID, err := primitive.ObjectIDFromHex(idStr)
			if err != nil {
				http.Error(w, "Formato de ID inválido para MongoDB", http.StatusBadRequest)
				return
			}

			var tarjetaEncontrada Tarjeta
			err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&tarjetaEncontrada)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					http.Error(w, "Tarjeta no encontrada", http.StatusNotFound)
				} else {
					log.Printf("Error buscando por ID: %v", err)
					http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
				}
				return
			}

			tarjetaEncontrada.CalcularCredito()

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tarjetaEncontrada)

		case http.MethodPost:
			var t Tarjeta
			err := json.NewDecoder(r.Body).Decode(&t)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			fmt.Printf("[DB] Ejecutando INSERT para: %s\n", t.Nombre)

			var existente Tarjeta
			err = collection.FindOne(ctx, bson.M{"nombre": t.Nombre}).Decode(&existente)
			if err == nil {
				http.Error(w, "Ya existe una tarjeta con ese nombre", http.StatusConflict)
				return
			} else if err != mongo.ErrNoDocuments {
				log.Printf("Error verificando existencia: %v", err)
				http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
				return
			}

			_, err = collection.InsertOne(ctx, t)
			if err != nil {
				log.Printf("Error creando tarjeta en DB: %v", err)
				http.Error(w, "Error al crear en la base de datos", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("¡Creado con éxito!"))

		case http.MethodPut:
			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				http.Error(w, "El ID es obligatorio en la URL (?id=...) para actualizar", http.StatusBadRequest)
				return
			}

			objID, err := primitive.ObjectIDFromHex(idStr)
			if err != nil {
				http.Error(w, "Formato de ID inválido", http.StatusBadRequest)
				return
			}

			var t Tarjeta
			err = json.NewDecoder(r.Body).Decode(&t)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			fmt.Printf("[DB] Ejecutando UPDATE para: %s\n", t.Nombre)

			updateData := bson.M{
				"$set": bson.M{
					"nombre":     t.Nombre,
					"disponible": t.Disponible,
					"saldo":      t.Saldo,
					"apagar":     t.Apagar,
					"color":      t.Color,
					"credito":    t.Credito,
					"saldoAPago": t.SaldoAPago,
					"fechaAPago": t.FechaPago,
				},
			}

			_, err = collection.UpdateOne(ctx, bson.M{"_id": objID}, updateData)
			if err != nil {
				log.Printf("Error actualizando tarjeta en DB: %v", err)
				http.Error(w, "Error al actualizar la base de datos", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("¡Actualizado con éxito!"))

		case http.MethodDelete:
			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				http.Error(w, "El ID es obligatorio para eliminar", http.StatusBadRequest)
				return
			}

			objID, err := primitive.ObjectIDFromHex(idStr)
			if err != nil {
				http.Error(w, "Formato de ID inválido", http.StatusBadRequest)
				return
			}

			resultado, err := collection.DeleteOne(ctx, bson.M{"_id": objID})
			if err != nil {
				log.Printf("Error eliminando tarjeta en DB: %v", err)
				http.Error(w, "Error al eliminar en la base de datos", http.StatusInternalServerError)
				return
			}

			if resultado.DeletedCount == 0 {
				http.Error(w, "Tarjeta no encontrada", http.StatusNotFound)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("¡Eliminado con éxito!"))
		default:
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		}
	}
}
