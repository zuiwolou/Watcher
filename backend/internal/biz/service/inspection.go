package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"watcher/backend/internal/biz/internal/model"
	"watcher/backend/pkg/bus"
	"watcher/backend/pkg/errcode"
)

func (s *Service) CreateInspectionPlan(ctx context.Context, name, cron string) (*model.InspectionPlan, error) {
	p := &model.InspectionPlan{Name: name, CycleCron: cron, Enabled: true}
	if err := s.db.WithContext(ctx).Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) CreateInspectionRoute(ctx context.Context, planID uint64, name string, forceOrder bool) (*model.InspectionRoute, error) {
	r := &model.InspectionRoute{PlanID: planID, Name: name, ForceOrder: forceOrder}
	if err := s.db.WithContext(ctx).Create(r).Error; err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Service) AddInspectionPoint(ctx context.Context, routeID uint64, seq int, name string, lat, lng, radius float64, requirePhoto bool) (*model.InspectionPoint, error) {
	p := &model.InspectionPoint{
		RouteID: routeID, Seq: seq, Name: name, Lat: lat, Lng: lng,
		RadiusM: radius, RequirePhoto: requirePhoto,
	}
	if err := s.db.WithContext(ctx).Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) CreateInspectionTask(ctx context.Context, planID, routeID, assigneeID uint64, dueAt *time.Time) (*model.InspectionTask, error) {
	t := &model.InspectionTask{
		PlanID: planID, RouteID: routeID, AssigneeID: assigneeID,
		Status: model.InspOpen, DueAt: dueAt,
	}
	if err := s.db.WithContext(ctx).Create(t).Error; err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) ListInspectionTasks(ctx context.Context, page, pageSize int) ([]model.InspectionTask, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	var total int64
	q := s.db.WithContext(ctx).Model(&model.InspectionTask{})
	_ = q.Count(&total)
	var list []model.InspectionTask
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

type InspectionCheckinInput struct {
	TaskID          uint64
	PointID         uint64
	Status          string // done|abnormal
	Lat, Lng        *float64
	PhotoKey        string
	Remark          string
	ClientRequestID string
	OperatorID      uint64
}

func (s *Service) InspectionCheckin(ctx context.Context, in InspectionCheckinInput) (*model.InspectionCheckin, *model.Alarm, error) {
	if in.ClientRequestID == "" {
		return nil, nil, fmt.Errorf("%s: clientRequestId required", errcode.ValidationError)
	}
	if in.Status != model.PointDone && in.Status != model.PointAbnormal {
		return nil, nil, fmt.Errorf("%s: status", errcode.ValidationError)
	}
	key := hashKey("insp_checkin:" + in.ClientRequestID)
	if hit, err := replayed(s.db.WithContext(ctx), key); err != nil {
		return nil, nil, err
	} else if hit {
		var ck model.InspectionCheckin
		if err := s.db.WithContext(ctx).Where("client_request_id = ?", in.ClientRequestID).First(&ck).Error; err == nil {
			return &ck, nil, nil
		}
	}

	var (
		ck  *model.InspectionCheckin
		alm *model.Alarm
	)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.InspectionTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, in.TaskID).Error; err != nil {
			return err
		}
		if task.Status == model.InspCompleted {
			return fmt.Errorf("%s: task already completed", errcode.StateInvalid)
		}
		var route model.InspectionRoute
		if err := tx.First(&route, task.RouteID).Error; err != nil {
			return err
		}
		var point model.InspectionPoint
		if err := tx.First(&point, in.PointID).Error; err != nil {
			return err
		}
		if point.RouteID != task.RouteID {
			return fmt.Errorf("%s: point not in route", errcode.StateInvalid)
		}
		// 同一点禁止重复打卡
		var dup int64
		if err := tx.Model(&model.InspectionCheckin{}).Where("task_id = ? AND point_id = ?", task.ID, point.ID).Count(&dup).Error; err != nil {
			return err
		}
		if dup > 0 {
			return fmt.Errorf("%s: point already checked in", errcode.StateInvalid)
		}
		// 强制点序：必须按 seq 打下一个未完成点
		if route.ForceOrder {
			var points []model.InspectionPoint
			if err := tx.Where("route_id = ?", task.RouteID).Order("seq asc, id asc").Find(&points).Error; err != nil {
				return err
			}
			var doneIDs []uint64
			if err := tx.Model(&model.InspectionCheckin{}).Where("task_id = ?", task.ID).Pluck("point_id", &doneIDs).Error; err != nil {
				return err
			}
			done := map[uint64]struct{}{}
			for _, id := range doneIDs {
				done[id] = struct{}{}
			}
			var next *model.InspectionPoint
			for i := range points {
				if _, ok := done[points[i].ID]; !ok {
					next = &points[i]
					break
				}
			}
			if next == nil {
				return fmt.Errorf("%s: all points already done", errcode.StateInvalid)
			}
			if next.ID != point.ID {
				return fmt.Errorf("%s: must check in point seq=%d first", errcode.StateInvalid, next.Seq)
			}
		}
		if point.RequirePhoto && in.PhotoKey == "" {
			return fmt.Errorf("%s: photo required", errcode.ValidationError)
		}
		// 有半径时定位必填，不可跳过半径校验
		if point.RadiusM > 0 {
			if in.Lat == nil || in.Lng == nil {
				return fmt.Errorf("%s: lat/lng required for radius check", errcode.ValidationError)
			}
			d := haversineM(point.Lat, point.Lng, *in.Lat, *in.Lng)
			if d > point.RadiusM {
				return fmt.Errorf("%s: out of point radius", errcode.StateInvalid)
			}
		}
		rec := &model.InspectionCheckin{
			TaskID: in.TaskID, PointID: in.PointID, Status: in.Status,
			Lat: in.Lat, Lng: in.Lng, PhotoKey: in.PhotoKey, Remark: in.Remark,
			ClientRequestID: in.ClientRequestID,
		}
		if err := tx.Create(rec).Error; err != nil {
			return err
		}
		if task.Status == model.InspOpen {
			task.Status = model.InspInProgress
			if err := tx.Save(&task).Error; err != nil {
				return err
			}
		}
		if in.Status == model.PointAbnormal {
			a := &model.Alarm{
				WarehouseID: "", RuleCode: "inspection_abnormal", Level: "important",
				Title: fmt.Sprintf("巡检异常: %s", point.Name), Status: model.AlarmRaised,
				RaisedAt: time.Now(), DetailJSON: in.Remark,
			}
			if err := tx.Create(a).Error; err != nil {
				return err
			}
			if err := s.enqueueBizOutbox(tx, "alarm.raised", "alarm", fmt.Sprintf("%d", a.ID), map[string]any{
				"alarmId": a.ID, "ruleCode": a.RuleCode, "title": a.Title, "taskId": task.ID, "pointId": point.ID,
			}); err != nil {
				return err
			}
			alm = a
		}
		// complete if all points have checkin
		var pointCnt int64
		tx.Model(&model.InspectionPoint{}).Where("route_id = ?", task.RouteID).Count(&pointCnt)
		var ckCnt int64
		tx.Model(&model.InspectionCheckin{}).Where("task_id = ?", task.ID).Distinct("point_id").Count(&ckCnt)
		if pointCnt > 0 && ckCnt >= pointCnt {
			task.Status = model.InspCompleted
			if err := tx.Save(&task).Error; err != nil {
				return err
			}
		}
		if err := claimIdempotency(tx, key, fmt.Sprintf("%d", rec.ID)); err != nil {
			return err
		}
		ck = rec
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if alm != nil {
		s.publish(bus.TopicAlarm, "svc-biz", map[string]any{
			"alarmId": alm.ID, "ruleCode": alm.RuleCode, "title": alm.Title, "level": alm.Level,
		})
	}
	return ck, alm, nil
}

func haversineM(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * R * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func (s *Service) ListInspectionPlans(ctx context.Context) ([]model.InspectionPlan, error) {
	var list []model.InspectionPlan
	err := s.db.WithContext(ctx).Find(&list).Error
	return list, err
}

func (s *Service) ListInspectionPoints(ctx context.Context, routeID uint64) ([]model.InspectionPoint, error) {
	var list []model.InspectionPoint
	err := s.db.WithContext(ctx).Where("route_id = ?", routeID).Order("seq asc").Find(&list).Error
	return list, err
}
