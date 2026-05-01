package modelos

import (
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Tarjeta struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Nombre          string             `bson:"nombre" json:"nombre"`
	Disponible      float64            `bson:"disponible" json:"disponible"`
	Saldo           float64            `bson:"saldo" json:"saldo"`
	Apagar          float64            `bson:"apagar" json:"apagar"`
	FechaPago       string             `bson:"fechaAPago" json:"fechaAPago"` // Se guarda como fechaAPago en Mongo
	Color           string             `bson:"color" json:"color"`
	Credito         float64            `bson:"credito" json:"credito"`
	SaldoAPago      float64            `bson:"saldoAPago" json:"saldoAPago"`
	SemanaAPago     int                `bson:"semanaAPago" json:"semanaAPago"`
	TenerAPago      float64            `bson:"tenerAPago" json:"tenerAPago"`
	SemanaCorriente int                `bson:"semanaCorriente" json:"semanaCorriente"`
	TenerCorriente  float64            `bson:"tenerCorriente" json:"tenerCorriente"`
	Tener           float64            `bson:"tener" json:"tener"`
	Apalancamiento  float64            `bson:"apalancamiento" json:"apalancamiento"`
	Msi             float64            `bson:"msi" json:"msi"`
	Uso             float64            `bson:"uso" json:"uso"`
	UsoPorcentaje   float64            `bson:"usoPorcentaje" json:"usoPorcentaje"`
}

func (t *Tarjeta) CalcularCredito() {
	t.Apagar = t.SaldoAPago

	fechaPago, err := time.Parse("2006-01-02", t.FechaPago)
	if err != nil {
		fechaPago, err = time.Parse("02/01/2006", t.FechaPago)
		if err != nil {
			fechaPago = time.Now()
		}
	}

	hoy := time.Now().Truncate(24 * time.Hour)
	diasAlViernes := (int(fechaPago.Weekday()) - 4 + 7) % 7
	inicioSemana7 := fechaPago.AddDate(0, 0, -diasAlViernes)
	diasDiff := int(inicioSemana7.Sub(hoy).Hours() / 24)

	semanas := 7
	if diasDiff > 0 {
		semanas = 7 - int(math.Ceil(float64(diasDiff)/7.0))
	}
	t.SemanaAPago = int(math.Max(1, math.Min(7, float64(semanas))))

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

	t.FechaPago = fechaPago.Format("02/01/2006")
}
