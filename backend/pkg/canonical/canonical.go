package canonical

// Message is the internal Canonical model (hw.canonical.v1).
type Message struct {
	Ver         uint32             `json:"ver"`
	MsgID       string             `json:"msgId"`
	MsgType     string             `json:"msgType"` // telemetry|location|event|heartbeat
	TsUnixMs    int64              `json:"tsUnixMs"`
	WarehouseID string             `json:"warehouseId"`
	DeviceID    string             `json:"deviceId"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
	Custom      map[string]any     `json:"custom,omitempty"`
	Location    *Location          `json:"location,omitempty"`
}

type Location struct {
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Speed    float64 `json:"speed"`
	Heading  float64 `json:"heading"`
	Accuracy float64 `json:"accuracy"`
	UnitID   string  `json:"unitId"`
}
