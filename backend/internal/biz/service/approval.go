package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"gorm.io/gorm"

	"watcher/backend/pkg/errcode"
	"watcher/backend/internal/biz/internal/model"
)

type ApprovalLevel struct {
	Level     int      `json:"level"`
	RoleCodes []string `json:"roleCodes"`
	UserIDs   []uint64 `json:"userIds"`
	Strategy  string   `json:"strategy"` // any|all
}

func (s *Service) UpsertApprovalTemplate(ctx context.Context, bizType, name string, levels []ApprovalLevel, enabled bool) (*model.ApprovalTemplate, error) {
	b, _ := json.Marshal(levels)
	var t model.ApprovalTemplate
	err := s.db.WithContext(ctx).Where("biz_type = ?", bizType).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		t = model.ApprovalTemplate{BizType: bizType, Name: name, LevelsJSON: string(b), Enabled: enabled}
		if err := s.db.Create(&t).Error; err != nil {
			return nil, err
		}
		return &t, nil
	}
	if err != nil {
		return nil, err
	}
	t.Name = name
	t.LevelsJSON = string(b)
	t.Enabled = enabled
	if err := s.db.Save(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Service) ListApprovalTemplates(ctx context.Context) ([]model.ApprovalTemplate, error) {
	var list []model.ApprovalTemplate
	err := s.db.WithContext(ctx).Find(&list).Error
	return list, err
}

func (s *Service) CreateStockAdjustment(ctx context.Context, warehouseID string, materialID uint64, delta float64, reason string, createdBy uint64) (*model.StockAdjustment, *model.ApprovalOrder, error) {
	var adj *model.StockAdjustment
	var order *model.ApprovalOrder
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		a := &model.StockAdjustment{
			WarehouseID: warehouseID, MaterialID: materialID, Delta: delta,
			Reason: reason, Status: model.AdjustPending, CreatedBy: createdBy,
		}
		if err := tx.Create(a).Error; err != nil {
			return err
		}
		var tpl model.ApprovalTemplate
		err := tx.Where("biz_type = ? AND enabled = ?", "stock_adjust", true).First(&tpl).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := applyStockAdjustment(tx, a); err != nil {
				return err
			}
			a.Status = model.AdjustApplied
			_ = tx.Save(a)
			adj = a
			return nil
		}
		if err != nil {
			return err
		}
		o := &model.ApprovalOrder{
			TemplateID: tpl.ID, BizType: "stock_adjust", BizID: strconv.FormatUint(a.ID, 10),
			Status: model.ApprovalPending, CurrentLevel: 1, ApplicantID: createdBy,
		}
		if err := tx.Create(o).Error; err != nil {
			return err
		}
		a.ApprovalOrderID = &o.ID
		_ = tx.Save(a)
		adj = a
		order = o
		return nil
	})
	return adj, order, err
}

func applyStockAdjustment(tx *gorm.DB, a *model.StockAdjustment) error {
	return adjustStock(tx, a.WarehouseID, a.MaterialID, a.Delta, "stock_adjust", strconv.FormatUint(a.ID, 10), "")
}

type ActApprovalInput struct {
	OrderID uint64
	ActorID uint64
	Action  string // pass|reject
	Comment string
	// HasRole reports whether actor has role code (injected from IAM)
	HasRole func(roleCode string) bool
}

func (s *Service) ActApproval(ctx context.Context, in ActApprovalInput) (*model.ApprovalOrder, error) {
	if in.Action != "pass" && in.Action != "reject" {
		return nil, fmt.Errorf("%s: action", errcode.ValidationError)
	}
	var out *model.ApprovalOrder
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var o model.ApprovalOrder
		if err := tx.First(&o, in.OrderID).Error; err != nil {
			return err
		}
		if o.Status != model.ApprovalPending {
			return fmt.Errorf("%s: order status", errcode.StateInvalid)
		}
		var tpl model.ApprovalTemplate
		if err := tx.First(&tpl, o.TemplateID).Error; err != nil {
			return err
		}
		var levels []ApprovalLevel
		_ = json.Unmarshal([]byte(tpl.LevelsJSON), &levels)
		cur := findLevel(levels, o.CurrentLevel)
		if cur == nil {
			return fmt.Errorf("%s: level config", errcode.StateInvalid)
		}
		if !actorAllowed(cur, in.ActorID, in.HasRole) {
			return fmt.Errorf("%s: not approver for level", errcode.AuthDenied)
		}
		act := &model.ApprovalAction{
			OrderID: o.ID, Level: o.CurrentLevel, ActorID: in.ActorID,
			Action: in.Action, Comment: in.Comment,
		}
		if err := tx.Create(act).Error; err != nil {
			return err
		}
		if in.Action == "reject" {
			o.Status = model.ApprovalRejected
			_ = tx.Save(&o)
			_ = rejectBiz(tx, o)
			out = &o
			return nil
		}
		maxLevel := maxLevelNum(levels)
		if o.CurrentLevel >= maxLevel {
			o.Status = model.ApprovalApproved
			_ = tx.Save(&o)
			if err := applyApprovedBiz(tx, o); err != nil {
				return err
			}
		} else {
			o.CurrentLevel++
			_ = tx.Save(&o)
		}
		out = &o
		return nil
	})
	return out, err
}

func findLevel(levels []ApprovalLevel, n int) *ApprovalLevel {
	for i := range levels {
		if levels[i].Level == n {
			return &levels[i]
		}
	}
	if n >= 1 && n <= len(levels) {
		return &levels[n-1]
	}
	return nil
}

func maxLevelNum(levels []ApprovalLevel) int {
	m := 0
	for _, l := range levels {
		if l.Level > m {
			m = l.Level
		}
	}
	if m == 0 {
		return len(levels)
	}
	return m
}

func actorAllowed(l *ApprovalLevel, actorID uint64, hasRole func(string) bool) bool {
	for _, id := range l.UserIDs {
		if id == actorID {
			return true
		}
	}
	if hasRole != nil {
		for _, rc := range l.RoleCodes {
			if hasRole(rc) {
				return true
			}
		}
	}
	// if no restriction configured, allow
	if len(l.UserIDs) == 0 && len(l.RoleCodes) == 0 {
		return true
	}
	return false
}

func rejectBiz(tx *gorm.DB, o model.ApprovalOrder) error {
	if o.BizType == "stock_adjust" {
		id, _ := strconv.ParseUint(o.BizID, 10, 64)
		return tx.Model(&model.StockAdjustment{}).Where("id = ?", id).Update("status", model.AdjustRejected).Error
	}
	return nil
}

func applyApprovedBiz(tx *gorm.DB, o model.ApprovalOrder) error {
	if o.BizType == "stock_adjust" {
		id, _ := strconv.ParseUint(o.BizID, 10, 64)
		var a model.StockAdjustment
		if err := tx.First(&a, id).Error; err != nil {
			return err
		}
		if err := applyStockAdjustment(tx, &a); err != nil {
			return err
		}
		a.Status = model.AdjustApplied
		return tx.Save(&a).Error
	}
	return nil
}

func (s *Service) ListApprovalTodo(ctx context.Context, page, pageSize int) ([]model.ApprovalOrder, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var total int64
	q := s.db.WithContext(ctx).Model(&model.ApprovalOrder{}).Where("status = ?", model.ApprovalPending)
	_ = q.Count(&total)
	var list []model.ApprovalOrder
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *Service) ListStockAdjustments(ctx context.Context, page, pageSize int) ([]model.StockAdjustment, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var total int64
	q := s.db.WithContext(ctx).Model(&model.StockAdjustment{})
	_ = q.Count(&total)
	var list []model.StockAdjustment
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}
