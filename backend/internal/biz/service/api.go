package service

import (
	"context"

	"watcher/backend/internal/biz/internal/model"
	"watcher/backend/pkg/jsid"
)

// BizAPI is the domain surface used by webhttp (BE-06).
// Today implemented in-process by *Service; tomorrow by gRPC client to cmd/biz.
type BizAPI interface {
	ListMaterials(ctx context.Context, page, pageSize int) ([]model.Material, int64, error)
	CreateMaterial(ctx context.Context, in CreateMaterialInput) (*model.Material, error)
	UpdateMaterial(ctx context.Context, id jsid.ID, in CreateMaterialInput) (*model.Material, error)
	ListAlarms(ctx context.Context, page, pageSize int, status, level string) ([]model.Alarm, int64, error)
	AckAlarm(ctx context.Context, id, actorID uint64) error
	CloseAlarm(ctx context.Context, id uint64) error
	EvaluateTelemetry(ctx context.Context, warehouseID, deviceID string, metrics map[string]float64) ([]model.Alarm, error)
	EvaluateLocation(ctx context.Context, warehouseID, deviceID string, lat, lng float64) ([]model.Alarm, error)
	ListUniqueCodes(ctx context.Context, warehouseID, status, q string, page, pageSize int) ([]model.UniqueCode, int64, error)
	VoidUniqueCode(ctx context.Context, code string, actorID uint64, reason string) (*model.UniqueCode, error)
	InspectionCheckin(ctx context.Context, in InspectionCheckinInput) (*model.InspectionCheckin, *model.Alarm, error)
	SubmitCheckin(ctx context.Context, in SubmitCheckinInput) (*model.TransportCheckin, error)
	ProcessOutboxBatch(ctx context.Context, limit int) (int, error)
}

// Compile-time check: *Service implements BizAPI.
var _ BizAPI = (*Service)(nil)
