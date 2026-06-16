package api

import (
	"encoding/json"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCalcularCredito_SoloImprimir(t *testing.T) {
	tarjetaTest := Tarjeta{
		ID:         primitive.ObjectID{},
		Nombre:     "Rappi",
		Credito:    59000,
		Disponible: 50578.14,
		Saldo:      5307.42,
		SaldoAPago: 2237.08,
		Color:      "ff5e3a",
		DiaCorte:   6,
		DiaPago:    27,
	}

	tarjetaTest.CalcularCredito()

	tarjetaJSON, err := json.MarshalIndent(tarjetaTest, "", "  ")
	if err != nil {
		t.Fatalf("Error al formatear tarjetaTest: %v", err)
	}

	t.Logf("Objeto completo:\n%s", string(tarjetaJSON))
}
