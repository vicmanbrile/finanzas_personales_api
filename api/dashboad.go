package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func TotalsHandler(mongoClient *mongo.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var totales struct {
			TotalCredito      float64                `json:"totalCredito"`
			TotalDisponible   float64                `json:"totalDisponible"`
			TotalAhorro       float64                `json:"totalAhorro"`
			TotalApalancado   float64                `json:"totalApalancado"`
			TotalMsi          float64                `json:"totalMsi"`
			TotalUsado        float64                `json:"totalUsado"`
			UtilizacionGlobal float64                `json:"utilizacionGlobal"`
			Q2Data            map[string]interface{} `json:"q2_data"`
			SaldoData         map[string]interface{} `json:"saldo_data"`
			Prestamos         []Prestamo             `json:"prestamos"`
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		db := mongoClient.Database(dbName)

		cursorTarjetas, err := db.Collection("tarjetas").Find(ctx, bson.D{})
		if err != nil {
			log.Printf("Error obteniendo tarjetas: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}
		defer cursorTarjetas.Close(ctx)

		var tarjetas []Tarjeta
		if err = cursorTarjetas.All(ctx, &tarjetas); err != nil {
			log.Printf("Error obteniendo totales de tarjetas: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}

		for i := range tarjetas {
			tarjetas[i].CalcularCredito()
			totales.TotalCredito += tarjetas[i].Credito
			totales.TotalDisponible += tarjetas[i].Disponible
			totales.TotalAhorro += tarjetas[i].Tener
			totales.TotalApalancado += tarjetas[i].Apalancamiento
			totales.TotalMsi += tarjetas[i].Msi
		}

		totales.TotalUsado = totales.TotalCredito - totales.TotalDisponible
		if totales.TotalCredito > 0 {
			totales.UtilizacionGlobal = (totales.TotalUsado / totales.TotalCredito) * 100
		}

		cursorPrestamos, err := db.Collection("prestamos").Find(ctx, bson.M{})
		if err != nil {
			log.Printf("Error obteniendo préstamos: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}
		defer cursorPrestamos.Close(ctx)

		var prestamosList []Prestamo
		if err = cursorPrestamos.All(ctx, &prestamosList); err != nil {
			log.Printf("Error decodificando préstamos: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}
		if prestamosList == nil {
			prestamosList = []Prestamo{}
		}

		var prestamosTotal float64
		for _, p := range prestamosList {
			prestamosTotal += p.Cantidad
		}

		cursorCuentas, err := db.Collection("cuentas").Find(ctx, bson.M{})
		if err != nil {
			log.Printf("Error obteniendo cuentas: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}
		defer cursorCuentas.Close(ctx)

		var cuentasList []CuentaDebito
		if err = cursorCuentas.All(ctx, &cuentasList); err != nil {
			log.Printf("Error decodificando cuentas: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}

		var debitoDisponible float64
		for _, c := range cuentasList {
			debitoDisponible += c.Saldo
		}
		arrastre := 7150.40
		presupuesto := 19230.00

		otrosGastosList := []float64{833.35, 1569.40, 1489.47, 396.99, 1451.09}
		var otrosGastos float64
		for _, gasto := range otrosGastosList {
			otrosGastos += gasto
		}

		ganado := 15000.00
		porGanar := 24000.00
		tarjetasTotal := 25352.42
		tarjetasAPagar := 5936.49

		q2 := arrastre + presupuesto + otrosGastos
		faltante := ganado - q2
		libre := faltante + porGanar

		pagado := ganado - debitoDisponible
		credito := tarjetasTotal - prestamosTotal
		diferido := credito - tarjetasAPagar
		solvencia := debitoDisponible - tarjetasAPagar

		totales.Q2Data = map[string]interface{}{
			"arrastre":     arrastre,
			"presupuesto":  presupuesto,
			"otros_gastos": otrosGastos,
			"q2_total":     q2,
			"faltante":     faltante,
			"libre":        libre,
		}

		totales.SaldoData = map[string]interface{}{
			"ganado":            ganado,
			"por_ganar":         porGanar,
			"debito_disponible": debitoDisponible,
			"pagado":            pagado,
			"prestamos_total":   prestamosTotal,
			"tarjetas_total":    tarjetasTotal,
			"tarjetas_a_pagar":  tarjetasAPagar,
			"credito":           credito,
			"diferido":          diferido,
			"solvencia":         solvencia,
		}
		totales.Prestamos = prestamosList
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(totales); err != nil {
			http.Error(w, "Error al generar el JSON", http.StatusInternalServerError)
		}
	}
}
