package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type DashboardResponse struct {
	Tarjetas []Tarjeta `json:"tarjetas"`
	Totals   struct {
		TotalCredito      float64 `json:"totalCredito"`
		TotalDisponible   float64 `json:"totalDisponible"`
		TotalAhorro       float64 `json:"totalAhorro"`
		TotalApalancado   float64 `json:"totalApalancado"`
		TotalMsi          float64 `json:"totalMsi"`
		TotalUsado        float64 `json:"totalUsado"`
		UtilizacionGlobal float64 `json:"utilizacionGlobal"`
	} `json:"totals"`
}

type Tarjeta struct {
	ID              int     `json:"id"`
	Nombre          string  `json:"nombre"`
	Color           string  `json:"color"`
	Credito         float64 `json:"credito"`
	SaldoAPago      float64 `json:"saldoAPago"`
	FechaAPago      string  `json:"fechaAPago"`
	SemanaAPago     int     `json:"semanaAPago"`
	TenerAPago      float64 `json:"tenerAPago"`
	SemanaCorriente int     `json:"semanaCorriente"`
	TenerCorriente  float64 `json:"tenerCorriente"`
	Tener           float64 `json:"tener"`
	Apalancamiento  float64 `json:"apalancamiento"`
	Msi             float64 `json:"msi"`
	Disponible      float64 `json:"disponible"`
	Uso             float64 `json:"uso"`
	UsoPorcentaje   float64 `json:"usoPorcentaje"`
}

func calcularTarjeta(row []string) (Tarjeta, error) {
	if len(row) < 6 {
		return Tarjeta{}, fmt.Errorf("fila incompleta")
	}

	nombre := row[0]
	credito, _ := strconv.ParseFloat(row[1], 64)
	disponible, _ := strconv.ParseFloat(row[2], 64)
	saldo, _ := strconv.ParseFloat(row[3], 64)
	fechaStr := strings.TrimSpace(row[4])
	saldoPagar, _ := strconv.ParseFloat(row[5], 64)
	color := "#6366f1"
	if len(row) > 6 && strings.TrimSpace(row[6]) != "" {
		color = strings.TrimSpace(row[6])
	}

	fechaPago, err := time.Parse("2006-01-02", fechaStr)
	if err != nil {
		fechaPago = time.Now()
	}

	// Lógica de semanas hasta el pago (basado en el viernes previo)
	hoy := time.Now().Truncate(24 * time.Hour)
	diasAlViernes := (int(fechaPago.Weekday()) - 4 + 7) % 7
	inicioSemana7 := fechaPago.AddDate(0, 0, -diasAlViernes)
	diasDiff := int(inicioSemana7.Sub(hoy).Hours() / 24)

	semanas := 7
	if diasDiff > 0 {
		semanas = 7 - int(math.Ceil(float64(diasDiff)/7.0))
	}
	semanas = int(math.Max(1, math.Min(7, float64(semanas))))

	// Cálculos financieros por tarjeta
	tenerAPago := math.Round((saldoPagar*float64(semanas)/7.0)*100) / 100
	saldoCorriente := math.Max(0.0, saldo-saldoPagar)

	semanaCorriente := 1
	if semanas > 4 {
		semanaCorriente = semanas - 4
	}
	tenerCorriente := math.Round((saldoCorriente*float64(semanaCorriente)/7.0)*100) / 100

	deudaTotal := math.Max(0.0, credito-disponible)
	msiTotal := math.Max(0.0, deudaTotal-saldo)
	tenerAcumulado := tenerCorriente + tenerAPago

	apalancamientoTotal := math.Max(0.0, deudaTotal-tenerAcumulado)
	msi := math.Min(msiTotal, apalancamientoTotal)
	apalancamientoNeto := math.Max(0.0, apalancamientoTotal-msi)

	usoPct := 0.0
	if credito > 0 {
		usoPct = math.Round((deudaTotal/credito*100)*10) / 10
	}

	return Tarjeta{
		// El ID se asigna en obtenerDatos() después de ordenar
		Nombre:          nombre,
		Color:           color,
		Credito:         credito,
		SaldoAPago:      saldoPagar,
		FechaAPago:      fechaPago.Format("02/01/2006"),
		SemanaAPago:     semanas,
		TenerAPago:      tenerAPago,
		SemanaCorriente: semanaCorriente,
		TenerCorriente:  tenerCorriente,
		Tener:           math.Min(deudaTotal, tenerAcumulado),
		Apalancamiento:  math.Round(apalancamientoNeto*100) / 100,
		Msi:             math.Round(msi*100) / 100,
		Disponible:      disponible,
		UsoPorcentaje:   usoPct,
	}, nil
}
