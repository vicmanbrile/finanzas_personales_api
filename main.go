package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	SheetURL  = "https://docs.google.com/spreadsheets/d/e/2PACX-1vTXdAW8zU6-897ZCb-1r4--VCALsGkuzo5psM4pimZhuaAqApY0gyKEvH6GtUgL0N5YwnqCfeTtpibj/pub?gid=0&single=true&output=csv"
	CacheFile = "cache_data.json"
	Port      = ":8000"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("static")))

	http.HandleFunc("/api/tarjetas", tarjetasHandler)
	http.HandleFunc("/api/tarjetas/", tarjetaByIDHandler)
	http.HandleFunc("/api/dashboard/totals", totalsHandler)

	fmt.Printf("Servidor Go corriendo en http://localhost%s\n", Port)
	log.Fatal(http.ListenAndServe(Port, nil))
}

// Función centralizada para leer, calcular y cachear los datos
func obtenerDatos() (DashboardResponse, error) {
	var response DashboardResponse

	resp, err := http.Get(SheetURL)
	if err != nil {
		// Si falla el CSV, intentamos leer la caché
		data, errCache := os.ReadFile(CacheFile)
		if errCache != nil {
			return response, errCache
		}
		json.Unmarshal(data, &response)
		return response, nil
	}
	defer resp.Body.Close()

	lector := csv.NewReader(resp.Body)
	lector.Read() // Saltar cabecera

	var tarjetas []Tarjeta

	for {
		record, err := lector.Read()
		if err == io.EOF {
			break
		}
		if t, err := calcularTarjeta(record); err == nil {
			tarjetas = append(tarjetas, t)
			response.Totals.TotalCredito += t.Credito
			response.Totals.TotalDisponible += t.Disponible
			response.Totals.TotalAhorro += t.Tener
			response.Totals.TotalApalancado += t.Apalancamiento
			response.Totals.TotalMsi += t.Msi
		}
	}

	response.Totals.TotalUsado = response.Totals.TotalCredito - response.Totals.TotalDisponible
	if response.Totals.TotalCredito > 0 {
		response.Totals.UtilizacionGlobal = (response.Totals.TotalUsado / response.Totals.TotalCredito) * 100
	}

	// Ordenamiento por prioridad
	sort.Slice(tarjetas, func(i, j int) bool {
		score := func(t Tarjeta) int {
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

	// Asignar ID basado en el orden una vez ordenadas
	for i := range tarjetas {
		tarjetas[i].ID = i
	}

	response.Tarjetas = tarjetas

	// Guardar en caché
	jsonData, _ := json.MarshalIndent(response, "", "    ")
	os.WriteFile(CacheFile, jsonData, 0644)

	return response, nil
}

// GET /api/v1/tarjetas
func tarjetasHandler(w http.ResponseWriter, r *http.Request) {
	// Redirigir si entra accidentalmente a /api/v1/tarjetas/ sin ID
	if r.URL.Path != "/api/tarjetas" {
		http.NotFound(w, r)
		return
	}

	datos, err := obtenerDatos()
	if err != nil {
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(datos.Tarjetas)
}

// GET /api/v1/tarjetas/{id}
func tarjetaByIDHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tarjetas/")
	if idStr == "" {
		http.NotFound(w, r)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido. Debe ser un número.", http.StatusBadRequest)
		return
	}

	datos, err := obtenerDatos()
	if err != nil {
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	// Validar que el ID exista en el arreglo
	if id < 0 || id >= len(datos.Tarjetas) {
		http.Error(w, "Tarjeta no encontrada", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(datos.Tarjetas[id])
}

// GET /api/v1/dashboard/totals
func totalsHandler(w http.ResponseWriter, r *http.Request) {
	datos, err := obtenerDatos()
	if err != nil {
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(datos.Totals)
}
