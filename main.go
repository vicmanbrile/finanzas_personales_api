package main

import (
	"context"
	"encoding/json"
	api "finanzas-personales/api"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	Port = ":8000"
)

var mongoClient *mongo.Client

func main() {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017/finanzas"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Error conectando a MongoDB:", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("No se pudo hacer ping a MongoDB:", err)
	}

	log.Println("Conectado exitosamente a MongoDB")
	mongoClient = client

	http.HandleFunc("/api/tarjetas", api.TarjetasHandler(mongoClient))
	http.HandleFunc("/api/dashboard/totals", api.TotalsHandler(mongoClient))
	http.HandleFunc("/api/finanzas", finanzasHandler)

	fmt.Printf("Servidor Go corriendo en http://localhost%s\n", Port)
	log.Fatal(http.ListenAndServe(Port, nil))
}

type Prestamo struct {
	Concepto string  `json:"concepto"`
	Cantidad float64 `json:"cantidad"`
	Saldado  float64 `json:"saldado"`
}

func finanzasHandler(w http.ResponseWriter, r *http.Request) {
	// ---------------------------------------------------------
	// 1. Datos Base y Arreglos (Q2_2026.txt y saldo.txt)
	// ---------------------------------------------------------
	arrastre := 7150.40
	presupuesto := 19230.00 // Total estático definido en el documento

	otrosGastosList := []float64{833.35, 1569.40, 1489.47, 396.99, 1451.09}
	var otrosGastos float64
	for _, gasto := range otrosGastosList {
		otrosGastos += gasto
	}

	prestamosList := []Prestamo{
		{Concepto: "Karla", Cantidad: 8487.50, Saldado: 0},
		{Concepto: "Victor", Cantidad: 297.90, Saldado: 0},
		{Concepto: "Viaje", Cantidad: 1075.00, Saldado: 0},
	}
	var prestamosTotal float64
	for _, p := range prestamosList {
		prestamosTotal += p.Cantidad
	}

	ganado := 15000.00
	porGanar := 24000.00
	debitoDisponible := 7013.15

	// ---------------------------------------------------------
	// 2. Datos de API Externa (Tarjetas)
	// ---------------------------------------------------------
	tarjetasTotal := 25352.42
	tarjetasAPagar := 5936.49

	// ---------------------------------------------------------
	// 3. Fórmulas y Cálculos
	// ---------------------------------------------------------

	// Cálculos de Q2
	q2 := arrastre + presupuesto + otrosGastos
	faltante := ganado - q2
	libre := faltante + porGanar

	// Cálculos de Saldo
	pagado := ganado - debitoDisponible
	credito := tarjetasTotal - prestamosTotal
	diferido := credito - tarjetasAPagar
	solvencia := debitoDisponible - tarjetasAPagar

	// ---------------------------------------------------------
	// 4. Construcción del JSON de Respuesta
	// ---------------------------------------------------------
	respuesta := map[string]interface{}{
		"q2_data": map[string]interface{}{
			"arrastre":     arrastre,
			"presupuesto":  presupuesto,
			"otros_gastos": otrosGastos, // Calculado: 5740.3
			"q2_total":     q2,          // Calculado: 32120.7
			"faltante":     faltante,    // Calculado: -17120.7
			"libre":        libre,       // Calculado: 6879.3
		},
		"saldo_data": map[string]interface{}{
			"ganado":            ganado,
			"por_ganar":         porGanar,
			"debito_disponible": debitoDisponible,
			"pagado":            pagado, // Calculado: 7986.85
			"prestamos_lista":   prestamosList,
			"prestamos_total":   prestamosTotal, // Calculado: 9860.4
			"tarjetas_total":    tarjetasTotal,  // Hardcodeado (Api Externa)
			"tarjetas_a_pagar":  tarjetasAPagar, // Hardcodeado (Api Externa)
			"credito":           credito,        // Calculado: 15492.02
			"diferido":          diferido,       // Calculado: 9555.53
			"solvencia":         solvencia,      // Calculado: 1076.66
		},
		"prestamos": prestamosList,
	}

	// Configurar headers y retornar respuesta
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(respuesta); err != nil {
		http.Error(w, "Error al generar el JSON", http.StatusInternalServerError)
	}
}
