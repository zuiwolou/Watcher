package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"watcher/backend/internal/biz/internal/model"
	"watcher/backend/pkg/bus"
	"watcher/backend/pkg/errcode"
)

func (s *Service) UpsertThreshold(ctx context.Context, warehouseID, metric string, upper, lower *float64) (*model.ThresholdRule, error) {
	var r model.ThresholdRule
	err := s.db.WithContext(ctx).Where("warehouse_id = ? AND metric = ?", warehouseID, metric).First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		r = model.ThresholdRule{WarehouseID: warehouseID, Metric: metric, Upper: upper, Lower: lower, Enabled: true, Version: 1}
		if err := s.db.Create(&r).Error; err != nil {
			return nil, err
		}
		return &r, nil
	}
	if err != nil {
		return nil, err
	}
	r.Upper, r.Lower = upper, lower
	r.Version++
	r.Enabled = true
	if err := s.db.Save(&r).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Service) ListThresholds(ctx context.Context, warehouseID string) ([]model.ThresholdRule, error) {
	var list []model.ThresholdRule
	q := s.db.WithContext(ctx).Model(&model.ThresholdRule{})
	if warehouseID != "" {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	err := q.Find(&list).Error
	return list, err
}

// alarmSuppressWindow 同一规则在同一设备上的告警抑制窗口，避免持续超阈时重复刷屏。
const alarmSuppressWindow = 10 * time.Minute

// EvaluateTelemetry compares metrics against thresholds and may raise alarms.
func (s *Service) EvaluateTelemetry(ctx context.Context, warehouseID, deviceID string, metrics map[string]float64) ([]model.Alarm, error) {
	var rules []model.ThresholdRule
	if err := s.db.WithContext(ctx).Where("warehouse_id = ? AND enabled = ?", warehouseID, true).Find(&rules).Error; err != nil {
		return nil, err
	}
	var raised []model.Alarm
	for _, r := range rules {
		v, ok := metrics[r.Metric]
		if !ok {
			continue
		}
		breach := false
		th := 0.0
		if r.Upper != nil && v > *r.Upper {
			breach, th = true, *r.Upper
		}
		if r.Lower != nil && v < *r.Lower {
			breach, th = true, *r.Lower
		}
		if !breach {
			continue
		}
		// suppress duplicate open alarm for same metric+device within window
		var cnt int64
		since := time.Now().Add(-alarmSuppressWindow)
		if err := s.db.WithContext(ctx).Model(&model.Alarm{}).Where(
			"warehouse_id = ? AND device_id = ? AND metric = ? AND status = ? AND raised_at > ?",
			warehouseID, deviceID, r.Metric, model.AlarmRaised, since).Count(&cnt).Error; err != nil {
			return raised, err
		}
		if cnt > 0 {
			continue
		}
		a := model.Alarm{
			WarehouseID: warehouseID, RuleCode: "threshold", Level: "important",
			Title: fmt.Sprintf("%s 超阈: %.2f", r.Metric, v), Status: model.AlarmRaised,
			Metric: r.Metric, Value: v, Threshold: th, DeviceID: deviceID, RaisedAt: time.Now(),
		}
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&a).Error; err != nil {
				return err
			}
			return s.enqueueBizOutbox(tx, "alarm.raised", "alarm", fmt.Sprintf("%d", a.ID), map[string]any{
				"alarmId": a.ID, "warehouseId": warehouseID, "deviceId": deviceID,
				"metric": a.Metric, "value": a.Value, "threshold": a.Threshold, "title": a.Title,
			})
		})
		if err != nil {
			return raised, err
		}
		raised = append(raised, a)
		s.publish(bus.TopicAlarm, "svc-biz", map[string]any{
			"alarmId": a.ID, "warehouseId": warehouseID, "deviceId": deviceID,
			"metric": a.Metric, "value": a.Value, "threshold": a.Threshold, "title": a.Title,
		})
	}
	return raised, nil
}

func (s *Service) ListAlarms(ctx context.Context, page, pageSize int, status string) ([]model.Alarm, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	var total int64
	q := s.db.WithContext(ctx).Model(&model.Alarm{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	_ = q.Count(&total)
	var list []model.Alarm
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *Service) AckAlarm(ctx context.Context, id, actorID uint64) error {
	now := time.Now()
	res := s.db.WithContext(ctx).Model(&model.Alarm{}).Where("id = ? AND status = ?", id, model.AlarmRaised).
		Updates(map[string]interface{}{"status": model.AlarmAcked, "acked_at": now, "acked_by": actorID})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%s: alarm", errcode.StateInvalid)
	}
	return nil
}

func (s *Service) CloseAlarm(ctx context.Context, id uint64) error {
	now := time.Now()
	res := s.db.WithContext(ctx).Model(&model.Alarm{}).Where("id = ? AND status IN ?", id, []string{model.AlarmRaised, model.AlarmAcked}).
		Updates(map[string]interface{}{"status": model.AlarmClosed, "closed_at": now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%s: alarm", errcode.StateInvalid)
	}
	return nil
}

type CircleGeo struct {
	Type    string  `json:"type"` // circle
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	RadiusM float64 `json:"radiusM"`
}

type PolygonGeo struct {
	Type   string       `json:"type"` // polygon
	Points [][2]float64 `json:"points"` // [[lat,lng], ...]
}

// pointInPolygon ray-casting; points are [lat, lng].
func pointInPolygon(lat, lng float64, points [][2]float64) bool {
	n := len(points)
	if n < 3 {
		return false
	}
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		yi, xi := points[i][0], points[i][1]
		yj, xj := points[j][0], points[j][1]
		if xi == xj {
			j = i
			continue // 竖直边不与水平射线稳定相交，跳过避免除零
		}
		intersect := ((xi > lng) != (xj > lng)) &&
			(lat < (yj-yi)*(lng-xi)/(xj-xi)+yi)
		if intersect {
			inside = !inside
		}
		j = i
	}
	return inside
}

func (s *Service) UpsertGeofence(ctx context.Context, id uint64, warehouseID, name, shapeType, geoJSON string, hysteresis float64, enabled bool) (*model.Geofence, error) {
	if shapeType == "" {
		shapeType = "circle"
	}
	if shapeType != "circle" && shapeType != "polygon" {
		return nil, fmt.Errorf("%s: shapeType must be circle|polygon", errcode.ValidationError)
	}
	if shapeType == "circle" {
		var c CircleGeo
		if err := json.Unmarshal([]byte(geoJSON), &c); err != nil || c.RadiusM <= 0 {
			return nil, fmt.Errorf("%s: circle geoJson requires lat/lng/radiusM", errcode.ValidationError)
		}
	}
	if shapeType == "polygon" {
		var p PolygonGeo
		if err := json.Unmarshal([]byte(geoJSON), &p); err != nil || len(p.Points) < 3 {
			return nil, fmt.Errorf("%s: polygon geoJson requires points[[lat,lng],...] >= 3", errcode.ValidationError)
		}
	}
	if id > 0 {
		var g model.Geofence
		if err := s.db.First(&g, id).Error; err != nil {
			return nil, err
		}
		g.Name, g.ShapeType, g.GeoJSON = name, shapeType, geoJSON
		g.HysteresisM, g.Enabled = hysteresis, enabled
		g.Version++
		if err := s.db.Save(&g).Error; err != nil {
			return nil, err
		}
		return &g, nil
	}
	g := &model.Geofence{
		WarehouseID: warehouseID, Name: name, ShapeType: shapeType, GeoJSON: geoJSON,
		HysteresisM: hysteresis, Enabled: enabled, Version: 1,
	}
	if err := s.db.Create(g).Error; err != nil {
		return nil, err
	}
	return g, nil
}

func (s *Service) ListGeofences(ctx context.Context, warehouseID string) ([]model.Geofence, error) {
	var list []model.Geofence
	q := s.db.WithContext(ctx).Model(&model.Geofence{})
	if warehouseID != "" {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	err := q.Find(&list).Error
	return list, err
}

// EvaluateLocation checks circle geofences; outside => alarm.
func (s *Service) EvaluateLocation(ctx context.Context, warehouseID, deviceID string, lat, lng float64) ([]model.Alarm, error) {
	var fences []model.Geofence
	if err := s.db.WithContext(ctx).Where("warehouse_id = ? AND enabled = ?", warehouseID, true).Find(&fences).Error; err != nil {
		return nil, err
	}
	var raised []model.Alarm
	for _, f := range fences {
		outside := false
		var distM float64
		switch f.ShapeType {
		case "", "circle":
			var c CircleGeo
			if err := json.Unmarshal([]byte(f.GeoJSON), &c); err != nil || c.RadiusM <= 0 {
				continue
			}
			distM = haversineM(c.Lat, c.Lng, lat, lng)
			outside = distM > c.RadiusM+f.HysteresisM
		case "polygon":
			var poly PolygonGeo
			if err := json.Unmarshal([]byte(f.GeoJSON), &poly); err != nil || len(poly.Points) < 3 {
				continue
			}
			inside := pointInPolygon(lat, lng, poly.Points)
			outside = !inside
			distM = 0
		default:
			continue // unsupported shape
		}
		if !outside {
			continue
		}
		var cnt int64
		since := time.Now().Add(-alarmSuppressWindow)
		if err := s.db.WithContext(ctx).Model(&model.Alarm{}).Where(
			"warehouse_id = ? AND device_id = ? AND rule_code = ? AND status = ? AND raised_at > ?",
			warehouseID, deviceID, fmt.Sprintf("geofence_%d", f.ID), model.AlarmRaised, since).Count(&cnt).Error; err != nil {
			return raised, err
		}
		if cnt > 0 {
			continue
		}
		a := model.Alarm{
			WarehouseID: warehouseID, RuleCode: fmt.Sprintf("geofence_%d", f.ID), Level: "urgent",
			Title: fmt.Sprintf("越界: %s", f.Name), Status: model.AlarmRaised,
			DeviceID: deviceID, RaisedAt: time.Now(),
			DetailJSON: fmt.Sprintf(`{"lat":%f,"lng":%f,"distanceM":%f,"shape":"%s"}`, lat, lng, distM, f.ShapeType),
		}
		if err := s.db.Create(&a).Error; err != nil {
			return raised, err
		}
		raised = append(raised, a)
	}
	return raised, nil
}
