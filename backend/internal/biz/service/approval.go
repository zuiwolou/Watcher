package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, in.OrderID).Error; err != nil {
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
		// 同一审批人在同一级只能表态一次，重复提交直接拒绝（需求 2.5：每级独立表态）。
		var dup int64
		if err := tx.Model(&model.ApprovalAction{}).
			Where("order_id = ? AND level = ? AND actor_id = ?", o.ID, o.CurrentLevel, in.ActorID).
			Count(&dup).Error; err != nil {
			return err
		}
		if dup > 0 {
			return fmt.Errorf("%s: already acted at level %d", errcode.StateInvalid, o.CurrentLevel)
		}
		if in.Action == "reject" && strings.TrimSpace(in.Comment) == "" {
			return fmt.Errorf("%s: reject requires comment", errcode.ValidationError)
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
			if err := tx.Save(&o).Error; err != nil {
				return err
			}
			if err := rejectBiz(tx, o); err != nil {
				return err
			}
			out = &o
			return nil
		}
		// 通过策略：all 需本级指定审批人全部表态，any 一人通过即可（需求 2.5）。
		if strategyAll(cur) && !levelPassed(tx, o.ID, o.CurrentLevel, requiredApprovers(cur)) {
			_ = tx.Save(&o)
			out = &o
			return nil
		}
		maxLevel := maxLevelNum(levels)
		if o.CurrentLevel >= maxLevel {
			o.Status = model.ApprovalApproved
			if err := tx.Save(&o).Error; err != nil {
				return err
			}
			if err := applyApprovedBiz(tx, o); err != nil {
				return err
			}
		} else {
			o.CurrentLevel++
			if err := tx.Save(&o).Error; err != nil {
				return err
			}
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

// actorMatched 严格匹配：仅当本级显式列出该用户或该用户拥有指定角色时为 true。
// 未配置审批人的级别不进入任何人的待办，避免待办列表对全员放开。
func actorMatched(l *ApprovalLevel, actorID uint64, hasRole func(string) bool) bool {
	for _, id := range l.UserIDs {
		if id == actorID {
			return true
		}
	}
	if hasRole == nil {
		return false
	}
	for _, rc := range l.RoleCodes {
		if hasRole(rc) {
			return true
		}
	}
	return false
}

// strategyAll 判断本级是否要求全员通过。
func strategyAll(l *ApprovalLevel) bool {
	return strings.EqualFold(l.Strategy, "all")
}

// requiredApprovers 本级要求的审批人数：显式用户 + 角色；未配置时为 1（至少一人）。
func requiredApprovers(l *ApprovalLevel) int {
	n := len(l.UserIDs) + len(l.RoleCodes)
	if n < 1 {
		return 1
	}
	return n
}

// levelPassed 统计本级已通过的不同审批人数，达到 required 才认为本级通过。
func levelPassed(tx *gorm.DB, orderID uint64, level, required int) bool {
	var n int64
	if err := tx.Model(&model.ApprovalAction{}).
		Where("order_id = ? AND level = ? AND action = ?", orderID, level, "pass").
		Distinct("actor_id").Count(&n).Error; err != nil {
		return false
	}
	return int(n) >= required
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
		// 副作用只执行一次：已过账的调整直接跳过，避免重复改库存（需求 7.6）。
		if a.Status == model.AdjustApplied {
			return nil
		}
		if err := applyStockAdjustment(tx, &a); err != nil {
			return err
		}
		a.Status = model.AdjustApplied
		return tx.Save(&a).Error
	}
	return nil
}

// ListApprovalTodo returns pending orders whose current level is actionable by the caller's roles.
func (s *Service) ListApprovalTodo(ctx context.Context, page, pageSize int, actorID uint64, hasRole func(string) bool) ([]model.ApprovalOrder, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	var all []model.ApprovalOrder
	if err := s.db.WithContext(ctx).Where("status = ?", model.ApprovalPending).Order("id desc").Find(&all).Error; err != nil {
		return nil, 0, err
	}
	// 模板一次性载入，避免逐单查询（N+1）。
	tplCache := map[uint64][]ApprovalLevel{}
	var tpls []model.ApprovalTemplate
	if err := s.db.WithContext(ctx).Find(&tpls).Error; err == nil {
		for _, t := range tpls {
			var lv []ApprovalLevel
			_ = json.Unmarshal([]byte(t.LevelsJSON), &lv)
			tplCache[t.ID] = lv
		}
	}
	filtered := make([]model.ApprovalOrder, 0, len(all))
	for _, o := range all {
		for _, lv := range tplCache[o.TemplateID] {
			if lv.Level != o.CurrentLevel {
				continue
			}
			if actorMatched(&lv, actorID, hasRole) {
				filtered = append(filtered, o)
				break
			}
		}
	}
	total := int64(len(filtered))
	start := (page - 1) * pageSize
	if start >= len(filtered) {
		return []model.ApprovalOrder{}, total, nil
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], total, nil
}

func (s *Service) ListStockAdjustments(ctx context.Context, page, pageSize int) ([]model.StockAdjustment, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	var total int64
	q := s.db.WithContext(ctx).Model(&model.StockAdjustment{})
	_ = q.Count(&total)
	var list []model.StockAdjustment
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *Service) ListApprovalOrders(ctx context.Context, page, pageSize int, status, bizType string) ([]model.ApprovalOrder, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	q := s.db.WithContext(ctx).Model(&model.ApprovalOrder{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if bizType != "" {
		q = q.Where("biz_type = ?", bizType)
	}
	var total int64
	_ = q.Count(&total)
	var list []model.ApprovalOrder
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

type ApprovalOrderDetail struct {
	Order    model.ApprovalOrder    `json:"order"`
	Actions  []model.ApprovalAction `json:"actions"`
	Template *model.ApprovalTemplate `json:"template,omitempty"`
	Levels   []ApprovalLevel        `json:"levels,omitempty"`
}

func (s *Service) GetApprovalOrder(ctx context.Context, id uint64) (*ApprovalOrderDetail, error) {
	var o model.ApprovalOrder
	if err := s.db.WithContext(ctx).First(&o, id).Error; err != nil {
		return nil, fmt.Errorf("%s: approval order", errcode.NotFound)
	}
	var acts []model.ApprovalAction
	_ = s.db.WithContext(ctx).Where("order_id = ?", id).Order("id asc").Find(&acts).Error
	detail := &ApprovalOrderDetail{Order: o, Actions: acts}
	var tpl model.ApprovalTemplate
	if err := s.db.WithContext(ctx).First(&tpl, o.TemplateID).Error; err == nil {
		detail.Template = &tpl
		_ = json.Unmarshal([]byte(tpl.LevelsJSON), &detail.Levels)
	}
	return detail, nil
}
