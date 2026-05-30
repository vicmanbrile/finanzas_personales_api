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

type CuentaDebito struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Nombre string  `json:"nombre" bson:"nombre"`
	Saldo  float64 `json:"saldo" bson:"saldo"`
}

func DebitsHandler(mongoClient *mongo.Client) http.HandlerFunc {
	var collectionName = "cuentas"

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/debito" {
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

				var cuenta CuentaDebito
				err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&cuenta)
				if err != nil {
					if err == mongo.ErrNoDocuments {
						http.Error(w, "Cuenta no encontrada", http.StatusNotFound)
					} else {
						log.Printf("Error obteniendo cuenta por ID: %v", err)
						http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
					}
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(cuenta)
				return
			}

			// Si no hay ID, obtenemos TODOS los documentos
			cursor, err := collection.Find(ctx, bson.M{})
			if err != nil {
				log.Printf("Error obteniendo cuentas: %v", err)
				http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
				return
			}
			defer cursor.Close(ctx)

			var cuentas []CuentaDebito
			if err = cursor.All(ctx, &cuentas); err != nil {
				log.Printf("Error decodificando cuentas: %v", err)
				http.Error(w, "Error al procesar los datos", http.StatusInternalServerError)
				return
			}

			if cuentas == nil {
				cuentas = []CuentaDebito{}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(cuentas)

		case http.MethodPost:
			var t CuentaDebito
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
			// Extraer ID del Query Param (?id=...)
			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				http.Error(w, "Falta el parámetro 'id' en la URL", http.StatusBadRequest)
				return
			}

			// Convertir string a ObjectID de Mongo
			objID, err := primitive.ObjectIDFromHex(idStr)
			if err != nil {
				http.Error(w, "Formato de ID inválido", http.StatusBadRequest)
				return
			}

			var t CuentaDebito
			if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Solo actualizamos los campos necesarios para no sobreescribir el _id
			update := bson.M{
				"$set": bson.M{
					"nombre": t.Nombre,
					"saldo":  t.Saldo,
				},
			}

			res, err := collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
			if err != nil {
				log.Printf("Error actualizando en DB: %v", err)
				http.Error(w, "Error interno al actualizar", http.StatusInternalServerError)
				return
			}

			if res.MatchedCount == 0 {
				http.Error(w, "Cuenta no encontrada", http.StatusNotFound)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("¡Actualizado con éxito!"))

		case http.MethodDelete:
			// Extraer ID del Query Param (?id=...)
			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				http.Error(w, "Falta el parámetro 'id' en la URL", http.StatusBadRequest)
				return
			}

			// Convertir string a ObjectID
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
				http.Error(w, "Cuenta no encontrada", http.StatusNotFound)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("¡Eliminado con éxito!"))

		default:
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		}
	}
}
