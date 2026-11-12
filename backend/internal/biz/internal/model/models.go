package model

import (
	"time"

	"watcher/backend/pkg/jsid"
)

// --- warehouse / material / stock ---

type Warehouse struct {
	ID        string `gorm:"primaryKey;size:64"`
	Code      string `gorm:"size:64;uniqueIndex"`
	Name      string `gorm:"size:128"`
	Address   string `gorm:"size:255"`
	Status    int8   `gorm:"default:1"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Warehouse) TableName() string { return "warehouses" }

type Material struct {
	ID        jsid.ID `gorm:"primaryKey"`
	Code      string  `gorm:"size:64;uniqueIndex"`
	Name      string  `gorm:"size:128"`
	WasteCode string  `gorm:"size:64"`
	Category  string  `gorm:"size:64"`
	Unit      string  `gorm:"size:16"`
	Spec      string  `gorm:"size:128"`
	Status    int8    `gorm:"default:1"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Material) TableName() string { return "materials" }

type StockBalance struct {
	ID          jsid.ID `gorm:"primaryKey"`
	WarehouseID string  `gorm:"size:64;uniqueIndex:uk_wh_mat"`
	MaterialID  jsid.ID `gorm:"uniqueIndex:uk_wh_mat"`
	Qty         float64 `gorm:"type:decimal(18,4);default:0"`
	LockedQty   float64 `gorm:"type:decimal(18,4);default:0"`
	UpdatedAt   time.Time
}

func (StockBalance) TableName() string { return "stock_balances" }

type StockLedger struct {
	ID           jsid.ID `gorm:"primaryKey"`
	WarehouseID  string  `gorm:"size:64;index"`
	MaterialID   jsid.ID `gorm:"index"`
	Delta        float64 `gorm:"type:decimal(18,4)"`
	BalanceAfter float64 `gorm:"type:decimal(18,4)"`
	BizType      string  `gorm:"size:32"`
	BizID        string  `gorm:"size:64"`
	UniqueCode   string  `gorm:"size:64"`
	CreatedAt    time.Time
}

func (StockLedger) TableName() string { return "stock_ledgers" }

// Unique code statuses
const (
	CodeIssued    = "issued"
	CodeInStock   = "in_stock"
	CodeLocked    = "locked"
	CodeInTransit = "in_transit"
	CodeSigned    = "signed"
	CodeVoided    = "voided"
)

type UniqueCode struct {
	ID              jsid.ID  `gorm:"primaryKey"`
	Code            string   `gorm:"size:64;uniqueIndex"`
	MaterialID      jsid.ID  `gorm:"index"`
	WarehouseID     string   `gorm:"size:64;index"`
	Status          string   `gorm:"size:16;index"`
	InboundOrderID  *jsid.ID
	OutboundOrderID *jsid.ID
	IssuedAt        *time.Time
	StockedAt       *time.Time
	SignedAt        *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (UniqueCode) TableName() string { return "unique_codes" }

// Order statuses
const (
	OrderDraft      = "draft"
	OrderOpen       = "open"
	OrderInProgress = "in_progress"
	OrderCompleted  = "completed"
	OrderCancelled  = "cancelled"
)

type InboundOrder struct {
	ID          jsid.ID `gorm:"primaryKey"`
	OrderNo     string  `gorm:"size:64;uniqueIndex"`
	WarehouseID string  `gorm:"size:64;index"`
	Status      string  `gorm:"size:16;index"`
	Remark      string  `gorm:"size:255"`
	ManifestNo  string  `gorm:"size:64"`
	CreatedBy   uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (InboundOrder) TableName() string { return "inbound_orders" }

type InboundOrderItem struct {
	ID             jsid.ID `gorm:"primaryKey"`
	InboundOrderID jsid.ID `gorm:"index"`
	MaterialID     jsid.ID
	Qty            float64 `gorm:"type:decimal(18,4)"`
}

func (InboundOrderItem) TableName() string { return "inbound_order_items" }

type InboundConfirmation struct {
	ID              jsid.ID `gorm:"primaryKey"`
	InboundOrderID  jsid.ID `gorm:"index"`
	UniqueCode      string  `gorm:"size:64;index"`
	OperatorID      uint64
	Lat             *float64
	Lng             *float64
	PhotoKey        string `gorm:"size:255"`
	ClientRequestID string `gorm:"size:64;uniqueIndex"`
	CreatedAt       time.Time
}

func (InboundConfirmation) TableName() string { return "inbound_confirmations" }

type OutboundOrder struct {
	ID          jsid.ID `gorm:"primaryKey"`
	OrderNo     string  `gorm:"size:64;uniqueIndex"`
	WarehouseID string  `gorm:"size:64;index"`
	Status      string  `gorm:"size:16;index"`
	Remark      string  `gorm:"size:255"`
	ManifestNo  string  `gorm:"size:64"`
	CreatedBy   uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (OutboundOrder) TableName() string { return "outbound_orders" }

type OutboundOrderItem struct {
	ID              jsid.ID `gorm:"primaryKey"`
	OutboundOrderID jsid.ID `gorm:"index"`
	MaterialID      jsid.ID
	Qty             float64 `gorm:"type:decimal(18,4)"`
}

func (OutboundOrderItem) TableName() string { return "outbound_order_items" }

type OutboundConfirmation struct {
	ID              jsid.ID `gorm:"primaryKey"`
	OutboundOrderID jsid.ID `gorm:"index"`
	UniqueCode      string  `gorm:"size:64;index"`
	Action          string  `gorm:"size:16"` // lock | receive
	OperatorID      uint64
	Lat             *float64
	Lng             *float64
	PhotoKey        string `gorm:"size:255"`
	ClientRequestID string `gorm:"size:64;uniqueIndex"`
	CreatedAt       time.Time
}

func (OutboundConfirmation) TableName() string { return "outbound_confirmations" }

type IdempotencyRecord struct {
	ID        uint64 `gorm:"primaryKey"`
	KeyHash   string `gorm:"size:128;uniqueIndex"`
	Response  string `gorm:"type:text"`
	CreatedAt time.Time
}

func (IdempotencyRecord) TableName() string { return "idempotency_records" }

type Alarm struct {
	ID          jsid.ID `gorm:"primaryKey"`
	WarehouseID string  `gorm:"size:64;index"`
	RuleCode    string  `gorm:"size:64"`
	Level       string  `gorm:"size:16"`
	Title       string  `gorm:"size:255"`
	Status      string  `gorm:"size:16;index"` // raised|acked|closed
	Metric      string  `gorm:"size:64"`
	Value       float64
	Threshold   float64
	DeviceID    string `gorm:"size:64"`
	RaisedAt    time.Time
	AckedAt     *time.Time
	AckedBy     *jsid.ID
	ClosedAt    *time.Time
	DetailJSON  string `gorm:"type:text"`
}

func (Alarm) TableName() string { return "alarms" }

// --- transport ---

const (
	TransportPending    = "pending"
	TransportInTransit  = "in_transit"
	TransportSigned     = "signed"
	CheckinDepart       = "depart"
	CheckinWaypoint     = "waypoint"
	CheckinArrive       = "arrive"
)

type TransportUnit struct {
	ID        uint64 `gorm:"primaryKey"`
	PlateNo   string `gorm:"size:64;uniqueIndex"`
	Type      string `gorm:"size:32"` // truck | transfer
	DeviceID  string `gorm:"size:64"`
	Status    int8   `gorm:"default:1"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (TransportUnit) TableName() string { return "transport_units" }

type TransportTask struct {
	ID              uint64 `gorm:"primaryKey"`
	TaskNo          string `gorm:"size:64;uniqueIndex"`
	OutboundOrderID jsid.ID `gorm:"index"`
	UnitID          uint64 `gorm:"index"`
	DriverUserID    uint64 `gorm:"index"`
	WarehouseID     string `gorm:"size:64;index"`
	Status          string `gorm:"size:16;index"`
	Remark          string `gorm:"size:255"`
	CreatedBy       uint64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (TransportTask) TableName() string { return "transport_tasks" }

type TransportCheckin struct {
	ID              uint64 `gorm:"primaryKey"`
	TaskID          uint64 `gorm:"index"`
	NodeType        string `gorm:"size:16"`
	UniqueCode      string `gorm:"size:64"`
	Lat             *float64
	Lng             *float64
	Accuracy        *float64
	PhotoKey        string `gorm:"size:255"`
	OperatorID      uint64
	ClientRequestID string `gorm:"size:64;uniqueIndex"`
	CreatedAt       time.Time
}

func (TransportCheckin) TableName() string { return "transport_checkins" }
