package store

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"watcher/backend/pkg/canonical"
)

type TelemetryRow struct {
	ID          uint64 `gorm:"primaryKey"`
	WarehouseID string `gorm:"size:64;index"`
	DeviceID    string `gorm:"size:64;index"`
	Metric      string `gorm:"size:64;index"`
	Value       float64
	Ts          time.Time `gorm:"index"`
	MsgID       string    `gorm:"size:64;uniqueIndex"`
}

func (TelemetryRow) TableName() string { return "hist_telemetry" }

type LocationRow struct {
	ID          uint64 `gorm:"primaryKey"`
	WarehouseID string `gorm:"size:64;index"`
	DeviceID    string `gorm:"size:64;index"`
	UnitID      string `gorm:"size:64"`
	Lat         float64
	Lng         float64
	Speed       float64
	Ts          time.Time `gorm:"index"`
	MsgID       string    `gorm:"size:64;uniqueIndex"`
}

func (LocationRow) TableName() string { return "hist_locations" }

type AlarmSnap struct {
	ID       uint64 `gorm:"primaryKey"`
	AlarmID  string `gorm:"size:64;index"`
	Status   string `gorm:"size:16"`
	Payload  string `gorm:"type:text"`
	Ts       time.Time
	MsgID    string `gorm:"size:64;uniqueIndex"`
}

func (AlarmSnap) TableName() string { return "hist_alarm_snapshots" }

type Store struct{ db *gorm.DB }

func New(db *gorm.DB) *Store { return &Store{db: db} }

func (s *Store) AutoMigrate() error {
	return s.db.AutoMigrate(&TelemetryRow{}, &LocationRow{}, &AlarmSnap{})
}

func (s *Store) IngestCanonical(ctx context.Context, m canonical.Message) error {
	ts := time.UnixMilli(m.TsUnixMs)
	if m.TsUnixMs == 0 {
		ts = time.Now()
	}
	switch m.MsgType {
	case "location":
		if m.Location == nil {
			return nil
		}
		return s.db.WithContext(ctx).Create(&LocationRow{
			WarehouseID: m.WarehouseID, DeviceID: m.DeviceID, UnitID: m.Location.UnitID,
			Lat: m.Location.Lat, Lng: m.Location.Lng, Speed: m.Location.Speed,
			Ts: ts, MsgID: m.MsgID + ":loc",
		}).Error
	default:
		for k, v := range m.Metrics {
			row := TelemetryRow{
				WarehouseID: m.WarehouseID, DeviceID: m.DeviceID, Metric: k, Value: v,
				Ts: ts, MsgID: m.MsgID + ":" + k,
			}
			if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
				// ignore duplicate msg
				continue
			}
		}
	}
	return nil
}

func (s *Store) QueryTelemetry(ctx context.Context, warehouseID, metric string, from, to time.Time, limit int) ([]TelemetryRow, error) {
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	q := s.db.WithContext(ctx).Where("warehouse_id = ? AND ts BETWEEN ? AND ?", warehouseID, from, to)
	if metric != "" {
		q = q.Where("metric = ?", metric)
	}
	var list []TelemetryRow
	err := q.Order("ts asc").Limit(limit).Find(&list).Error
	return list, err
}

func (s *Store) QueryTrack(ctx context.Context, warehouseID, deviceID string, from, to time.Time, limit int) ([]LocationRow, error) {
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	q := s.db.WithContext(ctx).Where("warehouse_id = ? AND ts BETWEEN ? AND ?", warehouseID, from, to)
	if deviceID != "" {
		q = q.Where("device_id = ?", deviceID)
	}
	var list []LocationRow
	err := q.Order("ts asc").Limit(limit).Find(&list).Error
	return list, err
}

func (s *Store) SaveAlarm(ctx context.Context, alarmID, status string, payload any, msgID string) error {
	b, _ := json.Marshal(payload)
	return s.db.WithContext(ctx).Create(&AlarmSnap{
		AlarmID: alarmID, Status: status, Payload: string(b), Ts: time.Now(), MsgID: msgID,
	}).Error
}
