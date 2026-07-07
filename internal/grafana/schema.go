package grafana

// rawFile decodes a Grafana dashboard export, which is either the dashboard
// object itself or wrapped as {"dashboard": {...}} (the API export form). The
// embedded rawDashboard captures the unwrapped case; the Dashboard pointer the
// wrapped one.
type rawFile struct {
	Dashboard *rawDashboard `json:"dashboard"`
	rawDashboard
}

func (rf rawFile) pick() rawDashboard {
	if rf.Dashboard != nil {
		return *rf.Dashboard
	}
	return rf.rawDashboard
}

type rawDashboard struct {
	Title   string     `json:"title"`
	Refresh string     `json:"refresh"`
	Time    rawTime    `json:"time"`
	Panels  []rawPanel `json:"panels"`
}

type rawTime struct {
	From string `json:"from"`
}

type rawPanel struct {
	Type        string         `json:"type"`
	Title       string         `json:"title"`
	GridPos     rawGridPos     `json:"gridPos"`
	Targets     []rawTarget    `json:"targets"`
	FieldConfig rawFieldConfig `json:"fieldConfig"`
	Options     rawOptions     `json:"options"`
	Panels      []rawPanel     `json:"panels"` // nested children of a collapsed row
}

type rawGridPos struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type rawTarget struct {
	Expr         string `json:"expr"`
	LegendFormat string `json:"legendFormat"`
	Instant      bool   `json:"instant"`
}

type rawFieldConfig struct {
	Defaults rawDefaults `json:"defaults"`
}

type rawDefaults struct {
	Unit       string        `json:"unit"`
	Decimals   *int          `json:"decimals"`
	Min        *float64      `json:"min"`
	Max        *float64      `json:"max"`
	Thresholds rawThresholds `json:"thresholds"`
	Custom     rawCustom     `json:"custom"`
}

type rawThresholds struct {
	Steps []rawStep `json:"steps"`
}

type rawStep struct {
	Color string   `json:"color"`
	Value *float64 `json:"value"`
}

type rawCustom struct {
	Stacking rawStacking `json:"stacking"`
}

type rawStacking struct {
	Mode string `json:"mode"`
}

type rawOptions struct {
	ReduceOptions rawReduce `json:"reduceOptions"`
	GraphMode     string    `json:"graphMode"` // stat panels: "area" (default) | "none"
}

type rawReduce struct {
	Calcs []string `json:"calcs"`
}
