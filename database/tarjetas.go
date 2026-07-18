package database

import (
	"context"
	"errors"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrTarjetaDuplicada = errors.New("ya existe una tarjeta con ese nombre")

type TarjetasRepository struct {
	collection *mongo.Collection
}

func (r *TarjetasRepository) All(ctx context.Context) ([]Tarjeta, error) {
	var tarjetas []Tarjeta

	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &tarjetas); err != nil {
		return nil, err
	}

	return tarjetas, nil
}

func (r *TarjetasRepository) FindOne(ctx context.Context, objID primitive.ObjectID) (Tarjeta, error) {
	var tarjeta Tarjeta

	err := r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&tarjeta)
	if err != nil {
		return Tarjeta{}, err
	}

	return tarjeta, nil
}

func (r *TarjetasRepository) Create(ctx context.Context, t *Tarjeta) error {
	var existente Tarjeta

	err := r.collection.FindOne(ctx, bson.M{"nombre": t.Nombre}).Decode(&existente)
	if err == nil {
		return ErrTarjetaDuplicada
	} else if err != mongo.ErrNoDocuments {
		return err
	}

	res, err := r.collection.InsertOne(ctx, t)
	if err != nil {
		return err
	}

	t.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *TarjetasRepository) Update(ctx context.Context, id primitive.ObjectID, t Tarjeta) error {
	updateData := bson.M{
		"$set": bson.M{
			"nombre":     t.Nombre,
			"disponible": t.Disponible,
			"saldo":      t.Saldo,
			"color":      t.Color,
			"credito":    t.Credito,
			"saldoAPago": t.SaldoAPago,
			"diaPago":    t.DiaPago,
			"diaCorte":   t.DiaCorte,
		},
	}

	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, updateData)
	return err
}

func (r *TarjetasRepository) Delete(ctx context.Context, id primitive.ObjectID) (int64, error) {
	resultado, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return 0, err
	}
	return resultado.DeletedCount, nil
}

type Tarjeta struct {
	// Datos guardados
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Nombre     string             `bson:"nombre" json:"nombre"`
	Credito    float64            `bson:"credito" json:"credito"`
	Disponible float64            `bson:"disponible" json:"disponible"`
	Saldo      float64            `bson:"saldo" json:"saldo"`
	SaldoAPago float64            `bson:"saldoAPago" json:"saldoAPago"`
	Color      string             `bson:"color" json:"color"`
	DiaCorte   int                `bson:"diaCorte" json:"diaCorte"`
	DiaPago    int                `bson:"diaPago" json:"diaPago"`
	// Estado del credito
	SemanaCorriente      int  `bson:"-" json:"semanaCorriente"`
	SemanaAPago          int  `bson:"-" json:"semanaAPago"`
	DiasParaProximoCorte int  `bson:"-" json:"diasParaProximoCorte"`
	DiasParaProximoPago  int  `bson:"-" json:"diasParaProximoPago"`
	TienePagoPendiente   bool `bson:"-" json:"tienePagoPendiente"`
	// Divicion del credito completo
	Uso            float64 `bson:"-" json:"uso"`
	UsoPorcentaje  float64 `bson:"-" json:"usoPorcentaje"`
	Tener          float64 `bson:"-" json:"tener"`
	Apalancamiento float64 `bson:"-" json:"apalancamiento"`
	Msi            float64 `bson:"-" json:"msi"`
	// Calculos de cuanto tener
	TenerAPago     float64 `bson:"-" json:"tenerAPago"`
	TenerCorriente float64 `bson:"-" json:"tenerCorriente"`
}

func (t *Tarjeta) CalcularCredito() {

	hoyTruncado := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)

	diaPago := t.DiaPago
	if diaPago == 0 {
		diaPago = 1
	}
	fechaPago := time.Date(hoyTruncado.Year(), hoyTruncado.Month(), diaPago, 0, 0, 0, 0, time.Local)
	if fechaPago.Before(hoyTruncado) {
		fechaPago = fechaPago.AddDate(0, 1, 0)
	}

	diaCorte := t.DiaCorte
	if diaCorte == 0 {
		diaCorte = 1
	}
	fechaCorte := time.Date(hoyTruncado.Year(), hoyTruncado.Month(), diaCorte, 0, 0, 0, 0, time.Local)
	if fechaCorte.Before(hoyTruncado) {
		fechaCorte = fechaCorte.AddDate(0, 1, 0)
	}

	diasAlViernes := (int(fechaPago.Weekday()) - 4 + 7) % 7
	inicioSemana7 := fechaPago.AddDate(0, 0, -diasAlViernes)
	diasDiff := int(inicioSemana7.Sub(hoyTruncado).Hours() / 24)

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

	t.Uso = math.Max(0.0, t.Credito-t.Disponible)
	msiTotal := math.Max(0.0, t.Uso-t.Saldo)
	tenerAcumulado := t.TenerCorriente + t.TenerAPago

	apalancamientoTotal := math.Max(0.0, t.Uso-tenerAcumulado)
	msi := math.Min(msiTotal, apalancamientoTotal)

	t.Tener = math.Min(t.Uso, tenerAcumulado)
	t.Msi = math.Round(msi*100) / 100
	t.Apalancamiento = math.Round(math.Max(0.0, apalancamientoTotal-msi)*100) / 100

	t.UsoPorcentaje = 0.0
	if t.Credito > 0 {
		t.UsoPorcentaje = math.Round((t.Uso/t.Credito*100)*10) / 10
	}

	corteTrun := fechaCorte
	pagoTrun := fechaPago

	t.TienePagoPendiente = t.SaldoAPago > 0

	pagoCorriente := pagoTrun
	if !pagoTrun.After(corteTrun) {
		pagoCorriente = time.Date(corteTrun.Year(), corteTrun.Month()+1, pagoTrun.Day(), 0, 0, 0, 0, time.Local)
	}

	calcularDias := func(destino time.Time) int {
		if !hoyTruncado.Before(destino) {
			return 0
		}
		return int(math.Round(destino.Sub(hoyTruncado).Hours() / 24.0))
	}

	t.DiasParaProximoCorte = calcularDias(corteTrun)
	t.DiasParaProximoPago = calcularDias(pagoCorriente)

}
