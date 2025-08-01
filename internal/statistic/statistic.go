package statistic

type Status string

const (
	STATUS_ERROR  Status = "Error"
	STATUS_NORMAL Status = "Normal"
)

type StatisticRow struct {
	Status
	Message string
}

func NewStatisticaRow(message string, status Status) *StatisticRow {
	return &StatisticRow{
		Status:  status,
		Message: message,
	}
}
