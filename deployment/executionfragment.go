package deployment

import "../metrics"

type ExecutionFragment struct {
	Cmd         string `json:"cmd"`
	Status      string
	StatusCmd   string `json:"status"`
	CheckCmd    string `json:"check"`
	ValidateCmd string `json:"validate"`
	metrics     *metrics.Metrics
}

type ExecutionFragments []*ExecutionFragment

func MakeExecutionFragment(def map[string]interface{}) (*ExecutionFragment, bool) {
	fragment := ExecutionFragment{}
	fragment.metrics = metrics.NewMetrics()
	// TODO error handling
	for key, val := range def {
		switch key {
		case "cmd":
			fragment.Cmd = val.(string)
		case "status":
			fragment.StatusCmd = val.(string)
		case "check":
			fragment.CheckCmd = val.(string)
		case "validate":
			fragment.ValidateCmd = val.(string)
		default:
			return nil, false
		}
	}
	return &fragment, true
}
