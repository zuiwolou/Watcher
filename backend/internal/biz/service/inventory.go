package service

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"watcher/backend/internal/biz/internal/domain"
	"watcher/backend/internal/biz/internal/model"
	"watcher/backend/pkg/errcode"
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

// PostInventory posts inventory differences against book stock (in_stock in warehouse).
// Missing codes: void + stock -1. Surplus scanned codes: recorded only (no stock increase).
func (s *Service) PostInventory(ctx context.Context, taskID uint64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.InventoryTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, taskID).Error; err != nil {
			return err
		}
		if task.Status != model.InvSubmitted {
			return fmt.Errorf("%s: expect submitted", errcode.StateInvalid)
		}

		var book []model.UniqueCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("warehouse_id = ? AND status = ?", task.WarehouseID, model.CodeInStock).
			Find(&book).Error; err != nil {
			return err
		}
		bookByCode := make(map[string]model.UniqueCode, len(book))
		for _, c := range book {
			bookByCode[c.Code] = c
		}

		var lines []model.InventoryLine
		if err := tx.Where("task_id = ? AND scanned = ?", taskID, true).Find(&lines).Error; err != nil {
			return err
		}
		scannedSet := make(map[string]model.InventoryLine, len(lines))
		for _, l := range lines {
			scannedSet[l.UniqueCode] = l
		}

		bizID := fmt.Sprintf("%d", task.ID)

		// Mark matched / surplus for scanned lines
		for code, line := range scannedSet {
			if _, ok := bookByCode[code]; ok {
				line.DiffType = model.DiffMatched
				line.Note = ""
			} else {
				line.DiffType = model.DiffSurplus
				line.Note = "scanned but not in_stock book; no stock increase"
			}
			if err := tx.Save(&line).Error; err != nil {
				return err
			}
		}

		// Missing: in book, not scanned → void + adjust stock
		for code, uc := range bookByCode {
			if _, ok := scannedSet[code]; ok {
				continue
			}
			if err := domain.CanTransit(uc.Status, model.CodeVoided); err != nil {
				return err
			}
			uc.Status = model.CodeVoided
			if err := tx.Save(&uc).Error; err != nil {
				return err
			}
			if err := adjustStock(tx, task.WarehouseID, uc.MaterialID, -1, "inventory_post", bizID, code); err != nil {
				return err
			}

			var existing model.InventoryLine
			err := tx.Where("task_id = ? AND unique_code = ?", taskID, code).First(&existing).Error
			if err == nil {
				existing.DiffType = model.DiffMissing
				existing.MaterialID = uc.MaterialID
				existing.Note = "book in_stock not scanned; voided on post"
				if err := tx.Save(&existing).Error; err != nil {
					return err
				}
			} else {
				miss := model.InventoryLine{
					TaskID: taskID, UniqueCode: code, MaterialID: uc.MaterialID,
					Scanned: false, DiffType: model.DiffMissing,
					Note: "book in_stock not scanned; voided on post",
				}
				if err := tx.Create(&miss).Error; err != nil {
					return err
				}
			}
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
