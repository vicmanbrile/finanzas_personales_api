package modelos

import (
	"math"
	"time"
)

type Tarjeta struct {
	ID              int     `json:"id"`
	Nombre          string  `json:"nombre"`
	Disponible      float64 `json:"disponible"`
	Saldo           float64 `json:"saldo"`
	Apagar          float64 `json:"apagar"`
	FechaPago       string  `json:"fechaAPago"`
	Color           string  `json:"color"`
	Credito         float64 `json:"credito"`
	SaldoAPago      float64 `json:"saldoAPago"`
	SemanaAPago     int     `json:"semanaAPago"`
	TenerAPago      float64 `json:"tenerAPago"`
	SemanaCorriente int     `json:"semanaCorriente"`
	TenerCorriente  float64 `json:"tenerCorriente"`
	Tener           float64 `json:"tener"`
	Apalancamiento  float64 `json:"apalancamiento"`
	Msi             float64 `json:"msi"`
	Uso             float64 `json:"uso"`
	UsoPorcentaje   float64 `json:"usoPorcentaje"`
}

func (t *Tarjeta) CalcularCredito() {
	// Sincronizar campos
	t.Apagar = t.SaldoAPago

	// CORRECCIÓN: Intentar parsear el formato de la base de datos (YYYY-MM-DD)
	// Si falla, intentar con el formato del frontend/formateado (DD/MM/YYYY)
	fechaPago, err := time.Parse("2006-01-02", t.FechaPago)
	if err != nil {
		fechaPago, err = time.Parse("02/01/2006", t.FechaPago)
		if err != nil {
			fechaPago = time.Now() // Solo cae aquí si la fecha viene vacía o totalmente rota
		}
	}

	// Lógica de semanas hasta el pago
	hoy := time.Now().Truncate(24 * time.Hour)
	diasAlViernes := (int(fechaPago.Weekday()) - 4 + 7) % 7
	inicioSemana7 := fechaPago.AddDate(0, 0, -diasAlViernes)
	diasDiff := int(inicioSemana7.Sub(hoy).Hours() / 24)

	semanas := 7
	if diasDiff > 0 {
		semanas = 7 - int(math.Ceil(float64(diasDiff)/7.0))
	}
	t.SemanaAPago = int(math.Max(1, math.Min(7, float64(semanas))))

	// Cálculos financieros
	t.TenerAPago = math.Round((t.SaldoAPago*float64(t.SemanaAPago)/7.0)*100) / 100

	saldoCorriente := math.Max(0.0, t.Saldo-t.SaldoAPago)

	t.SemanaCorriente = 1
	if t.SemanaAPago > 4 {
		t.SemanaCorriente = t.SemanaAPago - 4
	}
	t.TenerCorriente = math.Round((saldoCorriente*float64(t.SemanaCorriente)/7.0)*100) / 100

	deudaTotal := math.Max(0.0, t.Credito-t.Disponible)
	msiTotal := math.Max(0.0, deudaTotal-t.Saldo)
	tenerAcumulado := t.TenerCorriente + t.TenerAPago

	apalancamientoTotal := math.Max(0.0, deudaTotal-tenerAcumulado)
	msi := math.Min(msiTotal, apalancamientoTotal)

	t.Tener = math.Min(deudaTotal, tenerAcumulado)
	t.Msi = math.Round(msi*100) / 100
	t.Apalancamiento = math.Round(math.Max(0.0, apalancamientoTotal-msi)*100) / 100

	t.UsoPorcentaje = 0.0
	if t.Credito > 0 {
		t.UsoPorcentaje = math.Round((deudaTotal/t.Credito*100)*10) / 10
	}

	// Ahora es seguro formatearlo, porque la función ya soporta leer el formato "02/01/2006"
	t.FechaPago = fechaPago.Format("02/01/2006")
}
