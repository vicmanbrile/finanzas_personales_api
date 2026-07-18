package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func (server *API) TotalsHandler(w http.ResponseWriter, r *http.Request) {
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
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tarjetas, err := server.DB.Tarjetas.All(ctx)
	if err != nil {
		log.Printf("Error obteniendo tarjetas: %v", err)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(totales); err != nil {
		log.Printf("Error al generar el JSON: %v", err)
		http.Error(w, "Error al generar la respuesta", http.StatusInternalServerError)
	}
}
