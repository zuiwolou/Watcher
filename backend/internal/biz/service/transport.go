package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"watcher/backend/pkg/errcode"
	"watcher/backend/internal/biz/internal/domain"
	"watcher/backend/internal/biz/internal/model"
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
	OutboundOrderID uint64
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
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
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
	key := hashKey("transport_checkin:" + in.ClientRequestID)
	var existing model.IdempotencyRecord
	if err := s.db.Where("key_hash = ?", key).First(&existing).Error; err == nil {
		var ck model.TransportCheckin
		if e2 := s.db.Where("client_request_id = ?", in.ClientRequestID).First(&ck).Error; e2 == nil {
			return &ck, nil
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
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
		_ = tx.Create(&model.IdempotencyRecord{KeyHash: key, Response: fmt.Sprintf("%d", ck.ID)}).Error
		result = ck
		return nil
	})
	return result, err
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
