package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	dbName = "finanzas"
)

type Tarjeta struct {
	// Datos guardados
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Nombre     string             `bson:"nombre" json:"nombre"`
	Credito    float64            `bson:"credito" json:"credito"`
	Disponible float64            `bson:"disponible" json:"disponible"`
	Saldo      float64            `bson:"saldo" json:"saldo"`
	SaldoAPago float64            `bson:"saldoAPago" json:"saldoAPago"`
	Color      string             `bson:"color" json:"color"`
	DiaCorte   int                `bson:"diaCorte" json:"diaCorte"`
	DiaPago    int                `bson:"diaPago" json:"diaPago"`
	// Estado del credito
	SemanaCorriente      int  `bson:"-" json:"semanaCorriente"`
	SemanaAPago          int  `bson:"-" json:"semanaAPago"`
	DiasParaProximoCorte int  `bson:"-" json:"diasParaProximoCorte"`
	DiasParaProximoPago  int  `bson:"-" json:"diasParaProximoPago"`
	TienePagoPendiente   bool `bson:"-" json:"tienePagoPendiente"`
	// Divicion del credito completo
	Uso            float64 `bson:"-" json:"uso"`
	UsoPorcentaje  float64 `bson:"-" json:"usoPorcentaje"`
	Tener          float64 `bson:"-" json:"tener"`
	Apalancamiento float64 `bson:"-" json:"apalancamiento"`
	Msi            float64 `bson:"-" json:"msi"`
	// Calculos de cuanto tener
	TenerAPago     float64 `bson:"-" json:"tenerAPago"`
	TenerCorriente float64 `bson:"-" json:"tenerCorriente"`
}

func (t *Tarjeta) CalcularCredito() {

	hoyTruncado := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)

	diaPago := t.DiaPago
	if diaPago == 0 {
		diaPago = 1
	}
	fechaPago := time.Date(hoyTruncado.Year(), hoyTruncado.Month(), diaPago, 0, 0, 0, 0, time.Local)
	if fechaPago.Before(hoyTruncado) {
		fechaPago = fechaPago.AddDate(0, 1, 0)
	}

	diaCorte := t.DiaCorte
	if diaCorte == 0 {
		diaCorte = 1
	}
	fechaCorte := time.Date(hoyTruncado.Year(), hoyTruncado.Month(), diaCorte, 0, 0, 0, 0, time.Local)
	if fechaCorte.Before(hoyTruncado) {
		fechaCorte = fechaCorte.AddDate(0, 1, 0)
	}

	diasAlViernes := (int(fechaPago.Weekday()) - 4 + 7) % 7
	inicioSemana7 := fechaPago.AddDate(0, 0, -diasAlViernes)
	diasDiff := int(inicioSemana7.Sub(hoyTruncado).Hours() / 24)

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

	corteTrun := fechaCorte
	pagoTrun := fechaPago

	t.TienePagoPendiente = t.SaldoAPago > 0

	pagoCorriente := pagoTrun
	if !pagoTrun.After(corteTrun) {
		pagoCorriente = time.Date(corteTrun.Year(), corteTrun.Month()+1, pagoTrun.Day(), 0, 0, 0, 0, time.Local)
	}

	calcularDias := func(destino time.Time) int {
		if !hoyTruncado.Before(destino) {
			return 0
		}
		return int(math.Round(destino.Sub(hoyTruncado).Hours() / 24.0))
	}

	t.DiasParaProximoCorte = calcularDias(corteTrun)
	t.DiasParaProximoPago = calcularDias(pagoCorriente)

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
				cursor, err := collection.Find(ctx, bson.D{})
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

				sort.Slice(tarjetas, func(i, j int) bool {
					return tarjetas[i].DiasParaProximoPago > tarjetas[j].DiasParaProximoPago
				})

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
					"color":      t.Color,
					"credito":    t.Credito,
					"saldoAPago": t.SaldoAPago,
					"diaPago":    t.DiaPago,
					"diaCorte":   t.DiaCorte,
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
