package model

import "time"

// --- inventory ---

const (
	InvOpen      = "open"
	InvScanning  = "scanning"
	InvSubmitted = "submitted"
	InvPosted    = "posted"
)

type InventoryTask struct {
	ID          uint64 `gorm:"primaryKey"`
	Name        string `gorm:"size:128"`
	WarehouseID string `gorm:"size:64;index"`
	ScopeJSON   string `gorm:"type:text"`
	Status      string `gorm:"size:16;index"`
	CreatedBy   uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (InventoryTask) TableName() string { return "inventory_tasks" }

type InventoryLine struct {
	ID         uint64 `gorm:"primaryKey"`
	TaskID     uint64 `gorm:"index"`
	UniqueCode string `gorm:"size:64;index"`
	MaterialID uint64
	Scanned    bool
	ScannedAt  *time.Time
	OperatorID uint64
}

func (InventoryLine) TableName() string { return "inventory_lines" }

// --- approval ---

const (
	ApprovalPending  = "pending"
	ApprovalApproved = "approved"
	ApprovalRejected = "rejected"
)

type ApprovalTemplate struct {
	ID         uint64 `gorm:"primaryKey"`
	BizType    string `gorm:"size:64;index"` // stock_adjust
	Name       string `gorm:"size:128"`
	LevelsJSON string `gorm:"type:text"` // [{"level":1,"roleCodes":["approver"],"strategy":"any"}]
	Enabled    bool   `gorm:"default:true"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (ApprovalTemplate) TableName() string { return "approval_templates" }

type ApprovalOrder struct {
	ID           uint64 `gorm:"primaryKey"`
	TemplateID   uint64 `gorm:"index"`
	BizType      string `gorm:"size:64;index"`
	BizID        string `gorm:"size:64;index"`
	Status       string `gorm:"size:16;index"`
	CurrentLevel int
	ApplicantID  uint64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (ApprovalOrder) TableName() string { return "approval_orders" }

type ApprovalAction struct {
	ID        uint64 `gorm:"primaryKey"`
	OrderID   uint64 `gorm:"index"`
	Level     int
	ActorID   uint64
	Action    string `gorm:"size:16"` // pass|reject
	Comment   string `gorm:"size:512"`
	CreatedAt time.Time
}

func (ApprovalAction) TableName() string { return "approval_actions" }

const (
	AdjustPending  = "pending"
	AdjustApproved = "approved"
	AdjustRejected = "rejected"
	AdjustApplied  = "applied"
)

type StockAdjustment struct {
	ID               uint64  `gorm:"primaryKey"`
	WarehouseID      string  `gorm:"size:64;index"`
	MaterialID       uint64  `gorm:"index"`
	Delta            float64 `gorm:"type:decimal(18,4)"`
	Reason           string  `gorm:"size:255"`
	Status           string  `gorm:"size:16;index"`
	ApprovalOrderID  *uint64
	CreatedBy        uint64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (StockAdjustment) TableName() string { return "stock_adjustments" }

// --- inspection ---

const (
	InspOpen        = "open"
	InspInProgress  = "in_progress"
	InspCompleted   = "completed"
	PointPending    = "pending"
	PointDone       = "done"
	PointAbnormal   = "abnormal"
)

type InspectionPlan struct {
	ID        uint64 `gorm:"primaryKey"`
	Name      string `gorm:"size:128"`
	CycleCron string `gorm:"size:64"`
	Enabled   bool   `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (InspectionPlan) TableName() string { return "inspection_plans" }

type InspectionRoute struct {
	ID     uint64 `gorm:"primaryKey"`
	PlanID uint64 `gorm:"index"`
	Name   string `gorm:"size:128"`
}

func (InspectionRoute) TableName() string { return "inspection_routes" }

type InspectionPoint struct {
	ID           uint64  `gorm:"primaryKey"`
	RouteID      uint64  `gorm:"index"`
	Seq          int
	Name         string  `gorm:"size:128"`
	Lat          float64
	Lng          float64
	RadiusM      float64 `gorm:"type:decimal(10,2);default:50"`
	RequirePhoto bool
}

func (InspectionPoint) TableName() string { return "inspection_points" }

type InspectionTask struct {
	ID         uint64 `gorm:"primaryKey"`
	PlanID     uint64 `gorm:"index"`
	RouteID    uint64 `gorm:"index"`
	AssigneeID uint64 `gorm:"index"`
	Status     string `gorm:"size:16;index"`
	DueAt      *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (InspectionTask) TableName() string { return "inspection_tasks" }

type InspectionCheckin struct {
	ID              uint64 `gorm:"primaryKey"`
	TaskID          uint64 `gorm:"index"`
	PointID         uint64 `gorm:"index"`
	Status          string `gorm:"size:16"`
	Lat             *float64
	Lng             *float64
	PhotoKey        string `gorm:"size:255"`
	Remark          string `gorm:"size:512"`
	ClientRequestID string `gorm:"size:64;uniqueIndex"`
	CreatedAt       time.Time
}

func (InspectionCheckin) TableName() string { return "inspection_checkins" }

// --- threshold / geofence ---

type ThresholdRule struct {
	ID          uint64  `gorm:"primaryKey"`
	WarehouseID string  `gorm:"size:64;index"`
	Metric      string  `gorm:"size:64;index"`
	Upper       *float64
	Lower       *float64
	ExtraJSON   string `gorm:"type:text"`
	Version     int    `gorm:"default:1"`
	Enabled     bool   `gorm:"default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (ThresholdRule) TableName() string { return "threshold_rules" }

type Geofence struct {
	ID           uint64  `gorm:"primaryKey"`
	WarehouseID  string  `gorm:"size:64;index"`
	Name         string  `gorm:"size:128"`
	ShapeType    string  `gorm:"size:16"` // circle|polygon
	GeoJSON      string  `gorm:"type:text"`
	HysteresisM  float64 `gorm:"type:decimal(10,2);default:0"`
	ScheduleJSON string  `gorm:"type:text"`
	Version      int     `gorm:"default:1"`
	Enabled      bool    `gorm:"default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (Geofence) TableName() string { return "geofences" }

const (
	AlarmRaised = "raised"
	AlarmAcked  = "acked"
	AlarmClosed = "closed"
)
