package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"watcher/backend/internal/biz/internal/domain"
	"watcher/backend/internal/biz/internal/model"
	"watcher/backend/pkg/bus"
	"watcher/backend/pkg/canonical"
	"watcher/backend/pkg/errcode"
	"watcher/backend/pkg/jsid"
)

func (s *Service) CreateTransportUnit(ctx context.Context, plateNo, typ, deviceID string) (*model.TransportUnit, error) {
	u := &model.TransportUnit{PlateNo: plateNo, Type: typ, DeviceID: deviceID, Status: 1}
	if err := s.db.WithContext(ctx).Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Service) ListTransportUnits(ctx context.Context) ([]model.TransportUnit, error) {
	var list []model.TransportUnit
	err := s.db.WithContext(ctx).Order("id desc").Find(&list).Error
	return list, err
}

type CreateTransportTaskInput struct {
	OutboundOrderID jsid.ID
	UnitID          uint64
	DriverUserID    uint64
	WarehouseID     string
	Remark          string
	CreatedBy       uint64
}

func (s *Service) CreateTransportTask(ctx context.Context, in CreateTransportTaskInput) (*model.TransportTask, error) {
	var order model.OutboundOrder
	if err := s.db.WithContext(ctx).First(&order, in.OutboundOrderID).Error; err != nil {
		return nil, fmt.Errorf("%s: outbound order", errcode.NotFound)
	}
	var unit model.TransportUnit
	if err := s.db.WithContext(ctx).First(&unit, in.UnitID).Error; err != nil {
		return nil, fmt.Errorf("%s: transport unit", errcode.NotFound)
	}
	wh := in.WarehouseID
	if wh == "" {
		wh = order.WarehouseID
	}
	task := &model.TransportTask{
		TaskNo:          fmt.Sprintf("TR%s", time.Now().Format("20060102150405")),
		OutboundOrderID: in.OutboundOrderID,
		UnitID:          in.UnitID,
		DriverUserID:    in.DriverUserID,
		WarehouseID:     wh,
		Status:          model.TransportPending,
		Remark:          in.Remark,
		CreatedBy:       in.CreatedBy,
	}
	if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

func (s *Service) ListTransportTasks(ctx context.Context, page, pageSize int) ([]model.TransportTask, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	var total int64
	q := s.db.WithContext(ctx).Model(&model.TransportTask{})
	_ = q.Count(&total)
	var list []model.TransportTask
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

type SubmitCheckinInput struct {
	TaskID          uint64
	NodeType        string
	UniqueCode      string
	OperatorID      uint64
	ClientRequestID string
	Lat, Lng        *float64
	Accuracy        *float64
	PhotoKey        string
}

func (s *Service) SubmitCheckin(ctx context.Context, in SubmitCheckinInput) (*model.TransportCheckin, error) {
	if in.ClientRequestID == "" {
		return nil, fmt.Errorf("%s: clientRequestId required", errcode.ValidationError)
	}
	switch in.NodeType {
	case model.CheckinDepart, model.CheckinWaypoint, model.CheckinArrive:
	default:
		return nil, fmt.Errorf("%s: invalid nodeType", errcode.ValidationError)
	}
	// 发车/到达节点后端强制必拍（与 APP 一致）
	if (in.NodeType == model.CheckinDepart || in.NodeType == model.CheckinArrive) && in.PhotoKey == "" {
		return nil, fmt.Errorf("%s: photo required for %s", errcode.ValidationError, in.NodeType)
	}
	key := hashKey("transport_checkin:" + in.ClientRequestID)
	if hit, err := replayed(s.db.WithContext(ctx), key); err != nil {
		return nil, err
	} else if hit {
		var ck model.TransportCheckin
		if err := s.db.WithContext(ctx).Where("client_request_id = ?", in.ClientRequestID).First(&ck).Error; err == nil {
			return &ck, nil
		}
	}

	var result *model.TransportCheckin
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.TransportTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, in.TaskID).Error; err != nil {
			return err
		}
		if task.Status == model.TransportSigned {
			return fmt.Errorf("%s: task already signed", errcode.StateInvalid)
		}
		// optional: move unique code in_stock/locked -> in_transit on depart (does not sign)
		if in.UniqueCode != "" && in.NodeType == model.CheckinDepart {
			var code model.UniqueCode
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", in.UniqueCode).First(&code).Error; err == nil {
				if code.Status == model.CodeLocked || code.Status == model.CodeInStock {
					if err := domain.CanTransit(code.Status, model.CodeInTransit); err == nil {
						code.Status = model.CodeInTransit
						_ = tx.Save(&code).Error
					}
				}
			}
		}
		ck := &model.TransportCheckin{
			TaskID:          task.ID,
			NodeType:        in.NodeType,
			UniqueCode:      in.UniqueCode,
			Lat:             in.Lat,
			Lng:             in.Lng,
			Accuracy:        in.Accuracy,
			PhotoKey:        in.PhotoKey,
			OperatorID:      in.OperatorID,
			ClientRequestID: in.ClientRequestID,
		}
		if err := tx.Create(ck).Error; err != nil {
			return err
		}
		if task.Status == model.TransportPending && in.NodeType == model.CheckinDepart {
			task.Status = model.TransportInTransit
			if err := tx.Save(&task).Error; err != nil {
				return err
			}
		}
		if err := claimIdempotency(tx, key, fmt.Sprintf("%d", ck.ID)); err != nil {
			return err
		}
		result = ck
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Best-effort：发位置到 history + 围栏评估（不失败打卡）。
	if in.Lat != nil && in.Lng != nil {
		wh := ""
		var task model.TransportTask
		if e := s.db.WithContext(ctx).First(&task, in.TaskID).Error; e == nil {
			wh = task.WarehouseID
		}
		if wh == "" {
			wh = "WH01"
		}
		deviceID := in.UniqueCode
		if deviceID == "" {
			deviceID = fmt.Sprintf("checkin-%d", in.TaskID)
		}
		acc := 0.0
		if in.Accuracy != nil {
			acc = *in.Accuracy
		}
		s.publish(bus.TopicCanonical, "svc-biz", canonical.Message{
			Ver: 1, MsgID: uuid.NewString(), MsgType: "location",
			TsUnixMs: time.Now().UnixMilli(), WarehouseID: wh, DeviceID: deviceID,
			Location: &canonical.Location{Lat: *in.Lat, Lng: *in.Lng, Accuracy: acc, UnitID: fmt.Sprintf("task-%d", in.TaskID)},
			Custom:   map[string]any{"source": "transport_checkin", "nodeType": in.NodeType, "taskId": in.TaskID},
		})
		_, _ = s.EvaluateLocation(ctx, wh, deviceID, *in.Lat, *in.Lng)
	}
	return result, nil
}

func (s *Service) ListCheckins(ctx context.Context, taskID uint64) ([]model.TransportCheckin, error) {
	var list []model.TransportCheckin
	err := s.db.WithContext(ctx).Where("task_id = ?", taskID).Order("id asc").Find(&list).Error
	return list, err
}

// MarkTaskSigned is called after outbound receive when desired; optional helper.
func (s *Service) MarkTaskSignedByOutbound(ctx context.Context, outboundOrderID uint64) error {
	return s.db.WithContext(ctx).Model(&model.TransportTask{}).
		Where("outbound_order_id = ? AND status <> ?", outboundOrderID, model.TransportSigned).
		Update("status", model.TransportSigned).Error
}
