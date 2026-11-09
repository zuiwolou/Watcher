package service

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"watcher/backend/pkg/errcode"
	"watcher/backend/internal/biz/internal/model"
)

func (s *Service) CreateInventoryTask(ctx context.Context, name, warehouseID, scopeJSON string, createdBy uint64) (*model.InventoryTask, error) {
	t := &model.InventoryTask{
		Name: name, WarehouseID: warehouseID, ScopeJSON: scopeJSON,
		Status: model.InvOpen, CreatedBy: createdBy,
	}
	if err := s.db.WithContext(ctx).Create(t).Error; err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) ListInventoryTasks(ctx context.Context, page, pageSize int) ([]model.InventoryTask, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	var total int64
	q := s.db.WithContext(ctx).Model(&model.InventoryTask{})
	_ = q.Count(&total)
	var list []model.InventoryTask
	err := q.Order("id desc").Offset((page-1)*pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *Service) ScanInventory(ctx context.Context, taskID uint64, uniqueCode string, operatorID uint64, clientReqID string) (*model.InventoryLine, error) {
	if clientReqID == "" {
		return nil, fmt.Errorf("%s: clientRequestId required", errcode.ValidationError)
	}
	key := hashKey("inv_scan:" + clientReqID)
	if hit, err := replayed(s.db.WithContext(ctx), key); err != nil {
		return nil, err
	} else if hit {
		var line model.InventoryLine
		if err := s.db.WithContext(ctx).Where("task_id = ? AND unique_code = ?", taskID, uniqueCode).First(&line).Error; err == nil {
			return &line, nil
		}
	}

	var line *model.InventoryLine
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.InventoryTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, taskID).Error; err != nil {
			return err
		}
		if task.Status != model.InvOpen && task.Status != model.InvScanning {
			return fmt.Errorf("%s: task status %s", errcode.StateInvalid, task.Status)
		}
		var code model.UniqueCode
		if err := tx.Where("code = ?", uniqueCode).First(&code).Error; err != nil {
			return fmt.Errorf("%s: unique code", errcode.NotFound)
		}
		now := time.Now()
		var existingLine model.InventoryLine
		err := tx.Where("task_id = ? AND unique_code = ?", taskID, uniqueCode).First(&existingLine).Error
		if err == nil {
			existingLine.Scanned = true
			existingLine.ScannedAt = &now
			existingLine.OperatorID = operatorID
			if err := tx.Save(&existingLine).Error; err != nil {
				return err
			}
			line = &existingLine
		} else {
			l := model.InventoryLine{
				TaskID: taskID, UniqueCode: uniqueCode, MaterialID: code.MaterialID,
				Scanned: true, ScannedAt: &now, OperatorID: operatorID,
			}
			if err := tx.Create(&l).Error; err != nil {
				return err
			}
			line = &l
		}
		task.Status = model.InvScanning
		if err := tx.Save(&task).Error; err != nil {
			return err
		}
		return claimIdempotency(tx, key, uniqueCode)
	})
	return line, err
}

func (s *Service) SubmitInventory(ctx context.Context, taskID uint64) error {
	return s.db.WithContext(ctx).Model(&model.InventoryTask{}).
		Where("id = ? AND status IN ?", taskID, []string{model.InvOpen, model.InvScanning}).
		Update("status", model.InvSubmitted).Error
}

// PostInventory marks task posted. Difference posting against book stock is recorded as ledger note;
// for MVP: ensure all in-stock codes in warehouse not scanned could be reported; we only flip status.
func (s *Service) PostInventory(ctx context.Context, taskID uint64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.InventoryTask
		if err := tx.First(&task, taskID).Error; err != nil {
			return err
		}
		if task.Status != model.InvSubmitted {
			return fmt.Errorf("%s: expect submitted", errcode.StateInvalid)
		}
		task.Status = model.InvPosted
		return tx.Save(&task).Error
	})
}

func (s *Service) ListInventoryLines(ctx context.Context, taskID uint64) ([]model.InventoryLine, error) {
	var list []model.InventoryLine
	err := s.db.WithContext(ctx).Where("task_id = ?", taskID).Find(&list).Error
	return list, err
}
