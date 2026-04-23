package modelos

type Totales struct {
	TotalCredito      float64 `json:"totalCredito"`
	TotalDisponible   float64 `json:"totalDisponible"`
	TotalAhorro       float64 `json:"totalAhorro"`
	TotalApalancado   float64 `json:"totalApalancado"`
	TotalMsi          float64 `json:"totalMsi"`
	TotalUsado        float64 `json:"totalUsado"`
	UtilizacionGlobal float64 `json:"utilizacionGlobal"`
}
