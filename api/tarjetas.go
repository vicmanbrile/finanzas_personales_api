package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"finanzas-personales/api/db/modelos"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	dbName         = "finanzas"
	collectionName = "tarjetas"
	SheetURL       = "https://docs.google.com/spreadsheets/d/e/2PACX-1vTXdAW8zU6-897ZCb-1r4--VCALsGkuzo5psM4pimZhuaAqApY0gyKEvH6GtUgL0N5YwnqCfeTtpibj/pub?gid=0&single=true&output=csv"
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

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tarjetaEncontrada)

		case http.MethodPost:
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				http.Error(w, "Error al procesar el formulario multipart", http.StatusBadRequest)
				return
			}

			accion := r.FormValue("action")
			nombre := r.FormValue("nombre")

			switch accion {
			case "update":
				fmt.Printf("[DB] Ejecutando UPDATE para: %s\n", nombre)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("¡Actualizado con éxito!"))
			case "create":
				fmt.Printf("[DB] Ejecutando INSERT para: %s\n", nombre)
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte("¡Creado con éxito!"))
			default:
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte("No creado"))
			}

		default:
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		}
	}
}

// InyectarDatosDesdeCSV es la función que debes llamar desde tu main.go
// Lee el CSV, calcula, ordena, asigna IDs y guarda en MongoDB SOLO si el nombre no existe.
func InyectarDatosDesdeCSV(mongoClient *mongo.Client) error {
	log.Println("Iniciando proceso de inyección de datos desde CSV a MongoDB...")
	var tarjetas []modelos.Tarjeta

	resp, err := http.Get(SheetURL)
	if err != nil {
		return fmt.Errorf("error al obtener el CSV: %v", err)
	}
	defer resp.Body.Close()

	lector := csv.NewReader(resp.Body)
	lector.FieldsPerRecord = -1
	lector.Read() // Omitir el encabezado

	for {
		record, err := lector.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 6 { // Validación rápida de la fila
			continue
		}

		// 1. Parseo de datos desde el CSV
		nombre := record[0]
		credito, _ := strconv.ParseFloat(record[1], 64)
		disponible, _ := strconv.ParseFloat(record[2], 64)
		saldo, _ := strconv.ParseFloat(record[3], 64)
		fechaStr := strings.TrimSpace(record[4])
		saldoPagar, _ := strconv.ParseFloat(record[5], 64)

		color := "#6366f1" // Color por defecto
		if len(record) > 6 && strings.TrimSpace(record[6]) != "" {
			color = strings.TrimSpace(record[6])
		}

		// 2. Creación de la estructura con los datos base
		t := modelos.Tarjeta{
			Nombre:     nombre,
			Credito:    credito,
			Disponible: disponible,
			Saldo:      saldo,
			FechaPago:  fechaStr,
			SaldoAPago: saldoPagar,
			Color:      color,
		}

		// 3. Ejecutar los cálculos (la tarjeta se "auto-modifica")
		t.CalcularCredito()

		// 4. Agregar a la lista temporal
		tarjetas = append(tarjetas, t)
	}

	// 5. Ordenamiento personalizado original del CSV
	sort.Slice(tarjetas, func(i, j int) bool {
		score := func(t modelos.Tarjeta) int {
			if t.SemanaCorriente > 0 && t.SemanaAPago > 4 {
				return 3
			}
			if t.SemanaAPago == 1 {
				return 2
			}
			return -t.SemanaAPago
		}
		return score(tarjetas[i]) < score(tarjetas[j])
	})

	// 6. Asignación de IDs y guardado en MongoDB
	collection := mongoClient.Database(dbName).Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nuevosInsertados := 0
	for i := range tarjetas {
		// ELIMINADO: tarjetas[i].ID = i + 1

		// Verificamos por nombre para no duplicar
		filtro := bson.M{"nombre": tarjetas[i].Nombre}
		var resultado bson.M
		err := collection.FindOne(ctx, filtro).Decode(&resultado)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				// No existe, MongoDB le creará su propio _id automáticamente al insertar
				_, errInsert := collection.InsertOne(ctx, tarjetas[i])
				if errInsert != nil {
					log.Printf("Error insertando tarjeta %s: %v", tarjetas[i].Nombre, errInsert)
				} else {
					log.Printf("Tarjeta inyectada exitosamente: %s", tarjetas[i].Nombre)
					nuevosInsertados++
				}
			} else {
				log.Printf("Error al buscar la tarjeta %s: %v", tarjetas[i].Nombre, err)
			}
		} else {
			log.Printf("La tarjeta '%s' ya existe. Saltando.", tarjetas[i].Nombre)
		}
	}

	return nil
}
