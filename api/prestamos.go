package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Prestamo struct {
	ID       string  `json:"id,omitempty" bson:"_id,omitempty"`
	Concepto string  `json:"concepto" bson:"concepto"`
	Cantidad float64 `json:"cantidad" bson:"cantidad"`
	Saldado  float64 `json:"saldado" bson:"saldado"`
}

func PrestamosHandler(mongoClient *mongo.Client) http.HandlerFunc {
	var collectionName = "prestamos"

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/prestamo" {
			http.NotFound(w, r)
			return
		}

		// Asumiendo que dbName es una variable global en tu paquete.
		collection := mongoClient.Database(dbName).Collection(collectionName)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		switch r.Method {

		case http.MethodGet:
			idStr := r.URL.Query().Get("id")

			// Si hay un ID en la URL, buscamos solo ese registro
			if idStr != "" {
				objID, err := primitive.ObjectIDFromHex(idStr)
				if err != nil {
					http.Error(w, "Formato de ID inválido", http.StatusBadRequest)
					return
				}

				var prestamo Prestamo
				err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&prestamo)
				if err != nil {
					if err == mongo.ErrNoDocuments {
						http.Error(w, "Préstamo no encontrado", http.StatusNotFound)
					} else {
						log.Printf("Error obteniendo préstamo por ID: %v", err)
						http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
					}
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(prestamo)
				return
			}

			// Si no hay ID, obtenemos TODOS los documentos
			cursor, err := collection.Find(ctx, bson.M{})
			if err != nil {
				log.Printf("Error obteniendo préstamos: %v", err)
				http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
				return
			}
			defer cursor.Close(ctx)

			var prestamos []Prestamo
			if err = cursor.All(ctx, &prestamos); err != nil {
				log.Printf("Error decodificando préstamos: %v", err)
				http.Error(w, "Error al procesar los datos", http.StatusInternalServerError)
				return
			}

			if prestamos == nil {
				prestamos = []Prestamo{}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(prestamos)

		case http.MethodPost:
			var p Prestamo
			err := json.NewDecoder(r.Body).Decode(&p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			fmt.Printf("[DB] Ejecutando INSERT para: %s\n", p.Concepto)

			// Validamos que no exista ya un préstamo con el mismo concepto
			var existente Prestamo
			err = collection.FindOne(ctx, bson.M{"concepto": p.Concepto}).Decode(&existente)
			if err == nil {
				http.Error(w, "Ya existe un préstamo con ese concepto", http.StatusConflict)
				return
			} else if err != mongo.ErrNoDocuments {
				log.Printf("Error verificando existencia: %v", err)
				http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
				return
			}

			_, err = collection.InsertOne(ctx, p)
			if err != nil {
				log.Printf("Error creando préstamo en DB: %v", err)
				http.Error(w, "Error al crear en la base de datos", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("¡Préstamo creado con éxito!"))

		case http.MethodPut:
			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				http.Error(w, "Falta el parámetro 'id' en la URL", http.StatusBadRequest)
				return
			}

			objID, err := primitive.ObjectIDFromHex(idStr)
			if err != nil {
				http.Error(w, "Formato de ID inválido", http.StatusBadRequest)
				return
			}

			var p Prestamo
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			update := bson.M{
				"$set": bson.M{
					"concepto": p.Concepto,
					"cantidad": p.Cantidad,
					"saldado":  p.Saldado,
				},
			}

			res, err := collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
			if err != nil {
				log.Printf("Error actualizando en DB: %v", err)
				http.Error(w, "Error interno al actualizar", http.StatusInternalServerError)
				return
			}

			if res.MatchedCount == 0 {
				http.Error(w, "Préstamo no encontrado", http.StatusNotFound)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("¡Préstamo actualizado con éxito!"))

		case http.MethodDelete:
			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				http.Error(w, "Falta el parámetro 'id' en la URL", http.StatusBadRequest)
				return
			}

			objID, err := primitive.ObjectIDFromHex(idStr)
			if err != nil {
				http.Error(w, "Formato de ID inválido", http.StatusBadRequest)
				return
			}

			res, err := collection.DeleteOne(ctx, bson.M{"_id": objID})
			if err != nil {
				log.Printf("Error eliminando de DB: %v", err)
				http.Error(w, "Error al eliminar de la base de datos", http.StatusInternalServerError)
				return
			}

			if res.DeletedCount == 0 {
				http.Error(w, "Préstamo no encontrado", http.StatusNotFound)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("¡Préstamo eliminado con éxito!"))

		default:
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		}
	}
}
