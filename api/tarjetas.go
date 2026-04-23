package api

import (
	"encoding/csv"
	"encoding/json"
	"finanzas-personales/api/db/modelos"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

func TarjetasHandler(mongoClient *mongo.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tarjetas" {
			http.NotFound(w, r)
			return
		}

		switch r.Method {

		case http.MethodGet:
			tarjetas, err := obtenerCreditos()
			if err != nil {
				http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			idStr := r.URL.Query().Get("id")

			if idStr == "" {
				json.NewEncoder(w).Encode(tarjetas)
				return
			}

			id, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, "ID inválido. Debe ser un número.", http.StatusBadRequest)
				return
			}

			if id < 0 || id >= len(tarjetas) {
				http.Error(w, "Tarjeta no encontrada", http.StatusNotFound)
				return
			}

			json.NewEncoder(w).Encode(tarjetas[id])

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

func ObtenerTotales() (totales modelos.Totales, err error) {

	var tarjetas, _ = obtenerCreditos()

	for _, t := range tarjetas {
		totales.TotalCredito += t.Credito
		totales.TotalDisponible += t.Disponible
		totales.TotalAhorro += t.Tener
		totales.TotalApalancado += t.Apalancamiento
		totales.TotalMsi += t.Msi
	}

	totales.TotalUsado = totales.TotalCredito - totales.TotalDisponible
	if totales.TotalCredito > 0 {
		totales.UtilizacionGlobal = (totales.TotalUsado / totales.TotalCredito) * 100
	}

	return totales, nil
}

// Esta funcion queda pendiente a que la parte del get para mongodb

func obtenerCreditos() ([]modelos.Tarjeta, error) {
	var tarjetas []modelos.Tarjeta

	resp, _ := http.Get(SheetURL)

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

		// 4. Agregar a la lista
		tarjetas = append(tarjetas, t)
	}

	// 5. Ordenamiento personalizado
	// Ahora funciona perfecto porque t.SemanaAPago ya fue calculada arriba
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

	// 6. Asignación de IDs una vez ordenado
	for i := range tarjetas {
		tarjetas[i].ID = i + 1 // o solo `i` si prefieres que empiece en 0
	}

	return tarjetas, nil
}
