package api

import (
	"context"
	"encoding/json"
	"finanzas-personales/api/db/modelos"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	dbName         = "finanzas"
	collectionName = "tarjetas"
)

func TarjetasHandler(mongoClient *mongo.Client) http.HandlerFunc {
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
				findOptions.SetSort(bson.D{{Key: "saldoapago", Value: -1}})

				cursor, err := collection.Find(ctx, bson.D{}, findOptions)
				if err != nil {
					log.Printf("Error consultando MongoDB: %v", err)
					http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
					return
				}
				defer cursor.Close(ctx)

				var tarjetas []modelos.Tarjeta
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

			var tarjetaEncontrada modelos.Tarjeta
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
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				http.Error(w, "Error al procesar el formulario multipart", http.StatusBadRequest)
				return
			}

			accion := r.FormValue("action")
			nombre := r.FormValue("nombre")
			color := r.FormValue("color")
			fechaPago := r.FormValue("fechaPago")

			credito, _ := strconv.ParseFloat(r.FormValue("credito"), 64)
			disponible, _ := strconv.ParseFloat(r.FormValue("disponible"), 64)
			saldo, _ := strconv.ParseFloat(r.FormValue("saldo"), 64)
			saldoAPago, _ := strconv.ParseFloat(r.FormValue("saldoAPago"), 64)

			if color == "" {
				color = "#6366f1"
			}

			nuevaTarjeta := modelos.Tarjeta{
				Nombre:     nombre,
				Credito:    credito,
				Disponible: disponible,
				Saldo:      saldo,
				FechaPago:  fechaPago,
				SaldoAPago: saldoAPago,
				Color:      color,
			}

			switch accion {
			case "update":
				fmt.Printf("[DB] Ejecutando UPDATE para: %s\n", nombre)
				idStr := r.FormValue("id")
				if idStr == "" {
					http.Error(w, "El ID es obligatorio para actualizar", http.StatusBadRequest)
					return
				}

				objID, err := primitive.ObjectIDFromHex(idStr)
				if err != nil {
					http.Error(w, "Formato de ID inválido", http.StatusBadRequest)
					return
				}

				// Creamos el documento de actualización usando los campos de BSON de tu modelo
				// Nota: Los campos calculados se guardarán como vacíos/cero en la BD,
				// pero no importa porque los recalculas en el GET.
				updateData := bson.M{
					"$set": bson.M{
						"nombre":          nuevaTarjeta.Nombre,
						"disponible":      nuevaTarjeta.Disponible,
						"saldo":           nuevaTarjeta.Saldo,
						"apagar":          nuevaTarjeta.Apagar,
						"fechaAPago":      nuevaTarjeta.FechaPago,
						"color":           nuevaTarjeta.Color,
						"credito":         nuevaTarjeta.Credito,
						"saldoAPago":      nuevaTarjeta.SaldoAPago,
						"semanaAPago":     nuevaTarjeta.SemanaAPago,
						"tenerAPago":      nuevaTarjeta.TenerAPago,
						"semanaCorriente": nuevaTarjeta.SemanaCorriente,
						"tenerCorriente":  nuevaTarjeta.TenerCorriente,
						"tener":           nuevaTarjeta.Tener,
						"apalancamiento":  nuevaTarjeta.Apalancamiento,
						"msi":             nuevaTarjeta.Msi,
						"uso":             nuevaTarjeta.Uso,
						"usoPorcentaje":   nuevaTarjeta.UsoPorcentaje,
					},
				}

				// Ejecutamos la actualización
				_, err = collection.UpdateOne(ctx, bson.M{"_id": objID}, updateData)
				if err != nil {
					log.Printf("Error actualizando tarjeta en DB: %v", err)
					http.Error(w, "Error al actualizar la base de datos", http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte("¡Actualizado con éxito!"))

			case "create":
				fmt.Printf("[DB] Ejecutando INSERT para: %s\n", nombre)

				// Opcional: Validar que el nombre no exista ya
				var existente modelos.Tarjeta
				err := collection.FindOne(ctx, bson.M{"nombre": nuevaTarjeta.Nombre}).Decode(&existente)
				if err == nil {
					http.Error(w, "Ya existe una tarjeta con ese nombre", http.StatusConflict)
					return
				} else if err != mongo.ErrNoDocuments {
					log.Printf("Error verificando existencia: %v", err)
					http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
					return
				}

				// Insertamos en la BD. MongoDB creará el '_id' automáticamente.
				_, err = collection.InsertOne(ctx, nuevaTarjeta)
				if err != nil {
					log.Printf("Error creando tarjeta en DB: %v", err)
					http.Error(w, "Error al crear en la base de datos", http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusCreated)
				w.Write([]byte("¡Creado con éxito!"))

			default:
				http.Error(w, "Acción no reconocida", http.StatusBadRequest)
			}
		default:
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		}
	}
}
