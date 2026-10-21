package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"watcher/backend/pkg/errcode"
	"watcher/backend/internal/biz/internal/domain"
	"watcher/backend/internal/biz/internal/model"
)

type Service struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) AutoMigrate() error {
	return s.db.AutoMigrate(
		&model.Warehouse{},
		&model.Material{},
		&model.StockBalance{},
		&model.StockLedger{},
		&model.UniqueCode{},
		&model.InboundOrder{},
		&model.InboundOrderItem{},
		&model.InboundConfirmation{},
		&model.OutboundOrder{},
		&model.OutboundOrderItem{},
		&model.OutboundConfirmation{},
		&model.IdempotencyRecord{},
		&model.Alarm{},
		&model.TransportUnit{},
		&model.TransportTask{},
		&model.TransportCheckin{},
		&model.InventoryTask{},
		&model.InventoryLine{},
		&model.ApprovalTemplate{},
		&model.ApprovalOrder{},
		&model.ApprovalAction{},
		&model.StockAdjustment{},
		&model.InspectionPlan{},
		&model.InspectionRoute{},
		&model.InspectionPoint{},
		&model.InspectionTask{},
		&model.InspectionCheckin{},
		&model.ThresholdRule{},
		&model.Geofence{},
	)
}

func (s *Service) SeedWarehouse(id, code, name string) error {
	var n int64
	s.db.Model(&model.Warehouse{}).Where("id = ?", id).Count(&n)
	if n > 0 {
		return nil
	}
	return s.db.Create(&model.Warehouse{ID: id, Code: code, Name: name, Status: 1}).Error
}

func (s *Service) SeedDefaults() error {
	var n int64
	s.db.Model(&model.ApprovalTemplate{}).Where("biz_type = ?", "stock_adjust").Count(&n)
	if n == 0 {
		levels := []ApprovalLevel{
			{Level: 1, RoleCodes: []string{"admin"}, Strategy: "any"},
			{Level: 2, RoleCodes: []string{"admin"}, Strategy: "any"},
		}
		_, err := s.UpsertApprovalTemplate(context.Background(), "stock_adjust", "库存调整默认两级", levels, true)
		if err != nil {
			return err
		}
	}
	return nil
}

type CreateMaterialInput struct {
	Code      string
	Name      string
	WasteCode string
	Category  string
	Unit      string
	Spec      string
}

func (s *Service) CreateMaterial(ctx context.Context, in CreateMaterialInput) (*model.Material, error) {
	m := &model.Material{
		Code: in.Code, Name: in.Name, WasteCode: in.WasteCode,
		Category: in.Category, Unit: in.Unit, Spec: in.Spec, Status: 1,
	}
	if err := s.db.WithContext(ctx).Create(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Service) ListMaterials(ctx context.Context, page, pageSize int) ([]model.Material, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var total int64
	q := s.db.WithContext(ctx).Model(&model.Material{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Material
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

type CreateInboundInput struct {
	WarehouseID string
	Remark      string
	ManifestNo  string
	CreatedBy   uint64
	MaterialID  uint64
	Qty         float64
}

func (s *Service) CreateInboundOrder(ctx context.Context, in CreateInboundInput) (*model.InboundOrder, error) {
	order := &model.InboundOrder{
		OrderNo:     fmt.Sprintf("IN%s", time.Now().Format("20060102150405")),
		WarehouseID: in.WarehouseID,
		Status:      model.OrderOpen,
		Remark:      in.Remark,
		ManifestNo:  in.ManifestNo,
		CreatedBy:   in.CreatedBy,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		item := &model.InboundOrderItem{
			InboundOrderID: order.ID,
			MaterialID:     in.MaterialID,
			Qty:            in.Qty,
		}
		return tx.Create(item).Error
	})
	return order, err
}

func (s *Service) IssueCodes(ctx context.Context, orderID uint64, count int) ([]model.UniqueCode, error) {
	if count < 1 || count > 500 {
		return nil, fmt.Errorf("%s: count out of range", errcode.ValidationError)
	}
	var order model.InboundOrder
	if err := s.db.WithContext(ctx).First(&order, orderID).Error; err != nil {
		return nil, err
	}
	var item model.InboundOrderItem
	if err := s.db.WithContext(ctx).Where("inbound_order_id = ?", orderID).First(&item).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	codes := make([]model.UniqueCode, 0, count)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := 0; i < count; i++ {
			c := model.UniqueCode{
				Code:           fmt.Sprintf("HW%s%04d", time.Now().Format("150405"), i+1) + uuid.NewString()[:4],
				MaterialID:     item.MaterialID,
				WarehouseID:    order.WarehouseID,
				Status:         model.CodeIssued,
				InboundOrderID: &order.ID,
				IssuedAt:       &now,
			}
			if err := tx.Create(&c).Error; err != nil {
				return err
			}
			codes = append(codes, c)
		}
		return nil
	})
	return codes, err
}

type ConfirmInboundInput struct {
	OrderID         uint64
	UniqueCode      string
	OperatorID      uint64
	ClientRequestID string
	Lat, Lng        *float64
	PhotoKey        string
}

type ConfirmResult struct {
	UniqueCode string `json:"uniqueCode"`
	Status     string `json:"status"`
	Replayed   bool   `json:"replayed"`
}

func (s *Service) ConfirmInbound(ctx context.Context, in ConfirmInboundInput) (*ConfirmResult, error) {
	if in.ClientRequestID == "" {
		return nil, fmt.Errorf("%s: clientRequestId required", errcode.ValidationError)
	}
	key := hashKey("inbound_confirm:" + in.ClientRequestID)
	var existing model.IdempotencyRecord
	if err := s.db.Where("key_hash = ?", key).First(&existing).Error; err == nil {
		return &ConfirmResult{UniqueCode: in.UniqueCode, Status: model.CodeInStock, Replayed: true}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var result *ConfirmResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.InboundOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, in.OrderID).Error; err != nil {
			return err
		}
		if order.Status != model.OrderOpen && order.Status != model.OrderInProgress {
			return fmt.Errorf("%s: order status %s", errcode.StateInvalid, order.Status)
		}
		var code model.UniqueCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", in.UniqueCode).First(&code).Error; err != nil {
			return err
		}
		if code.InboundOrderID == nil || *code.InboundOrderID != order.ID {
			return fmt.Errorf("%s: code not in order", errcode.StateInvalid)
		}
		if err := domain.CanTransit(code.Status, model.CodeInStock); err != nil {
			return err
		}
		now := time.Now()
		code.Status = model.CodeInStock
		code.StockedAt = &now
		if err := tx.Save(&code).Error; err != nil {
			return err
		}
		conf := &model.InboundConfirmation{
			InboundOrderID:  order.ID,
			UniqueCode:      code.Code,
			OperatorID:      in.OperatorID,
			Lat:             in.Lat,
			Lng:             in.Lng,
			PhotoKey:        in.PhotoKey,
			ClientRequestID: in.ClientRequestID,
		}
		if err := tx.Create(conf).Error; err != nil {
			return err
		}
		if err := adjustStock(tx, order.WarehouseID, code.MaterialID, 1, "inbound_confirm", fmt.Sprintf("%d", order.ID), code.Code); err != nil {
			return err
		}
		order.Status = model.OrderInProgress
		var remain int64
		tx.Model(&model.UniqueCode{}).Where("inbound_order_id = ? AND status = ?", order.ID, model.CodeIssued).Count(&remain)
		if remain == 0 {
			order.Status = model.OrderCompleted
		}
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		_ = tx.Create(&model.IdempotencyRecord{KeyHash: key, Response: code.Code}).Error
		result = &ConfirmResult{UniqueCode: code.Code, Status: code.Status}
		return nil
	})
	return result, err
}

type LockOutboundInput struct {
	OrderID         uint64
	UniqueCode      string
	OperatorID      uint64
	ClientRequestID string
	Lat, Lng        *float64
	PhotoKey        string
}

func (s *Service) LockOutbound(ctx context.Context, in LockOutboundInput) (*ConfirmResult, error) {
	if in.ClientRequestID == "" {
		return nil, fmt.Errorf("%s: clientRequestId required", errcode.ValidationError)
	}
	key := hashKey("outbound_lock:" + in.ClientRequestID)
	var existing model.IdempotencyRecord
	if err := s.db.Where("key_hash = ?", key).First(&existing).Error; err == nil {
		return &ConfirmResult{UniqueCode: in.UniqueCode, Status: model.CodeLocked, Replayed: true}, nil
	}

	var result *ConfirmResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.OutboundOrder
		if err := tx.First(&order, in.OrderID).Error; err != nil {
			return err
		}
		var code model.UniqueCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", in.UniqueCode).First(&code).Error; err != nil {
			return err
		}
		if err := domain.CanTransit(code.Status, model.CodeLocked); err != nil {
			return err
		}
		code.Status = model.CodeLocked
		code.OutboundOrderID = &order.ID
		if err := tx.Save(&code).Error; err != nil {
			return err
		}
		if err := lockStock(tx, order.WarehouseID, code.MaterialID, 1); err != nil {
			return err
		}
		conf := &model.OutboundConfirmation{
			OutboundOrderID: order.ID,
			UniqueCode:      code.Code,
			Action:          "lock",
			OperatorID:      in.OperatorID,
			Lat:             in.Lat,
			Lng:             in.Lng,
			PhotoKey:        in.PhotoKey,
			ClientRequestID: in.ClientRequestID,
		}
		if err := tx.Create(conf).Error; err != nil {
			return err
		}
		order.Status = model.OrderInProgress
		_ = tx.Save(&order).Error
		_ = tx.Create(&model.IdempotencyRecord{KeyHash: key, Response: code.Code}).Error
		result = &ConfirmResult{UniqueCode: code.Code, Status: code.Status}
		return nil
	})
	return result, err
}

type ReceiveOutboundInput struct {
	OrderID         uint64
	UniqueCode      string
	OperatorID      uint64
	ClientRequestID string
	Lat, Lng        *float64
	PhotoKey        string
}

func (s *Service) ReceiveOutbound(ctx context.Context, in ReceiveOutboundInput) (*ConfirmResult, error) {
	if in.ClientRequestID == "" {
		return nil, fmt.Errorf("%s: clientRequestId required", errcode.ValidationError)
	}
	key := hashKey("outbound_receive:" + in.ClientRequestID)
	var existing model.IdempotencyRecord
	if err := s.db.Where("key_hash = ?", key).First(&existing).Error; err == nil {
		return &ConfirmResult{UniqueCode: in.UniqueCode, Status: model.CodeSigned, Replayed: true}, nil
	}

	var result *ConfirmResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.OutboundOrder
		if err := tx.First(&order, in.OrderID).Error; err != nil {
			return err
		}
		var code model.UniqueCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", in.UniqueCode).First(&code).Error; err != nil {
			return err
		}
		// locked or in_transit -> signed
		to := model.CodeSigned
		if err := domain.CanTransit(code.Status, to); err != nil {
			return err
		}
		now := time.Now()
		prev := code.Status
		code.Status = model.CodeSigned
		code.SignedAt = &now
		if err := tx.Save(&code).Error; err != nil {
			return err
		}
		if prev == model.CodeLocked {
			if err := consumeLockedStock(tx, order.WarehouseID, code.MaterialID, 1, fmt.Sprintf("%d", order.ID), code.Code); err != nil {
				return err
			}
		} else {
			if err := adjustStock(tx, order.WarehouseID, code.MaterialID, -1, "outbound_receive", fmt.Sprintf("%d", order.ID), code.Code); err != nil {
				return err
			}
		}
		conf := &model.OutboundConfirmation{
			OutboundOrderID: order.ID,
			UniqueCode:      code.Code,
			Action:          "receive",
			OperatorID:      in.OperatorID,
			Lat:             in.Lat,
			Lng:             in.Lng,
			PhotoKey:        in.PhotoKey,
			ClientRequestID: in.ClientRequestID,
		}
		if err := tx.Create(conf).Error; err != nil {
			return err
		}
		_ = tx.Create(&model.IdempotencyRecord{KeyHash: key, Response: code.Code}).Error
		_ = tx.Model(&model.TransportTask{}).
			Where("outbound_order_id = ? AND status <> ?", order.ID, model.TransportSigned).
			Update("status", model.TransportSigned)
		result = &ConfirmResult{UniqueCode: code.Code, Status: code.Status}
		return nil
	})
	return result, err
}

func (s *Service) CreateOutboundOrder(ctx context.Context, warehouseID string, createdBy, materialID uint64, qty float64, remark string) (*model.OutboundOrder, error) {
	order := &model.OutboundOrder{
		OrderNo:     fmt.Sprintf("OUT%s", time.Now().Format("20060102150405")),
		WarehouseID: warehouseID,
		Status:      model.OrderOpen,
		Remark:      remark,
		CreatedBy:   createdBy,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		return tx.Create(&model.OutboundOrderItem{
			OutboundOrderID: order.ID,
			MaterialID:      materialID,
			Qty:             qty,
		}).Error
	})
	return order, err
}

func (s *Service) GetUniqueCode(ctx context.Context, code string) (*model.UniqueCode, error) {
	var c model.UniqueCode
	err := s.db.WithContext(ctx).Where("code = ?", code).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%s: unique code", errcode.NotFound)
	}
	return &c, err
}

func (s *Service) ListStocks(ctx context.Context, warehouseID string) ([]model.StockBalance, error) {
	var list []model.StockBalance
	q := s.db.WithContext(ctx).Model(&model.StockBalance{})
	if warehouseID != "" {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	err := q.Find(&list).Error
	return list, err
}

func (s *Service) ListInboundOrders(ctx context.Context, page, pageSize int) ([]model.InboundOrder, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var total int64
	q := s.db.WithContext(ctx).Model(&model.InboundOrder{})
	_ = q.Count(&total)
	var list []model.InboundOrder
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func adjustStock(tx *gorm.DB, warehouseID string, materialID uint64, delta float64, bizType, bizID, uniqueCode string) error {
	var bal model.StockBalance
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("warehouse_id = ? AND material_id = ?", warehouseID, materialID).
		First(&bal).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		bal = model.StockBalance{WarehouseID: warehouseID, MaterialID: materialID, Qty: 0}
		if err := tx.Create(&bal).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	bal.Qty += delta
	if bal.Qty < 0 {
		return fmt.Errorf("%s: insufficient stock", errcode.StateInvalid)
	}
	if err := tx.Save(&bal).Error; err != nil {
		return err
	}
	return tx.Create(&model.StockLedger{
		WarehouseID: warehouseID, MaterialID: materialID, Delta: delta,
		BalanceAfter: bal.Qty, BizType: bizType, BizID: bizID, UniqueCode: uniqueCode,
	}).Error
}

func lockStock(tx *gorm.DB, warehouseID string, materialID uint64, qty float64) error {
	var bal model.StockBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("warehouse_id = ? AND material_id = ?", warehouseID, materialID).
		First(&bal).Error; err != nil {
		return err
	}
	avail := bal.Qty - bal.LockedQty
	if avail < qty {
		return fmt.Errorf("%s: insufficient available stock", errcode.StateInvalid)
	}
	bal.LockedQty += qty
	return tx.Save(&bal).Error
}

func consumeLockedStock(tx *gorm.DB, warehouseID string, materialID uint64, qty float64, bizID, uniqueCode string) error {
	var bal model.StockBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("warehouse_id = ? AND material_id = ?", warehouseID, materialID).
		First(&bal).Error; err != nil {
		return err
	}
	if bal.LockedQty < qty || bal.Qty < qty {
		return fmt.Errorf("%s: insufficient locked stock", errcode.StateInvalid)
	}
	bal.LockedQty -= qty
	bal.Qty -= qty
	if err := tx.Save(&bal).Error; err != nil {
		return err
	}
	return tx.Create(&model.StockLedger{
		WarehouseID: warehouseID, MaterialID: materialID, Delta: -qty,
		BalanceAfter: bal.Qty, BizType: "outbound_receive", BizID: bizID, UniqueCode: uniqueCode,
	}).Error
}

func hashKey(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
