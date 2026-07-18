package api

import (
	"context"
	"encoding/json"
	"finanzas-personales/database"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (server *API) TarjetasHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/tarjetas" {
		http.NotFound(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch r.Method {

	case http.MethodGet:
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			tarjetas, err := server.DB.Tarjetas.All(ctx)
			if err != nil {
				log.Printf("Error obteniendo resultados: %v", err)
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
			http.Error(w, "Formato de ID inválido", http.StatusBadRequest)
			return
		}

		tarjetaEncontrada, err := server.DB.Tarjetas.FindOne(ctx, objID)
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
		var t database.Tarjeta
		err := json.NewDecoder(r.Body).Decode(&t)
		if err != nil {
			http.Error(w, "Error decodificando el body: "+err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Printf("[DB] Ejecutando INSERT para: %s\n", t.Nombre)

		err = server.DB.Tarjetas.Create(ctx, &t)
		if err != nil {
			if err == database.ErrTarjetaDuplicada {
				http.Error(w, err.Error(), http.StatusConflict)
			} else {
				log.Printf("Error creando tarjeta en DB: %v", err)
				http.Error(w, "Error al crear en la base de datos", http.StatusInternalServerError)
			}
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

		var t database.Tarjeta
		err = json.NewDecoder(r.Body).Decode(&t)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Printf("[DB] Ejecutando UPDATE para: %s\n", t.Nombre)

		err = server.DB.Tarjetas.Update(ctx, objID, t)
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

		deletedCount, err := server.DB.Tarjetas.Delete(ctx, objID)
		if err != nil {
			log.Printf("Error eliminando tarjeta en DB: %v", err)
			http.Error(w, "Error al eliminar en la base de datos", http.StatusInternalServerError)
			return
		}

		if deletedCount == 0 {
			http.Error(w, "Tarjeta no encontrada", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("¡Eliminado con éxito!"))

	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}
