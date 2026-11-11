package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"watcher/backend/internal/biz/internal/domain"
	"watcher/backend/internal/biz/internal/model"
	"watcher/backend/pkg/errcode"
)

// EventPublisher is optional NSQ (or compatible) publisher for config/biz events.
type EventPublisher interface {
	Publish(topic string, producer string, payload any) error
}

type Service struct {
	db  *gorm.DB
	pub EventPublisher
}

func New(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) SetPublisher(p EventPublisher) { s.pub = p }

func (s *Service) publish(topic, producer string, payload any) {
	if s == nil || s.pub == nil {
		return
	}
	_ = s.pub.Publish(topic, producer, payload)
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
		&model.MqttSchemaProfile{},
		&model.IntegrationConfig{},
		&model.OpenCredential{},
		&model.Device{},
		&model.IntegrationOutbox{},
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

func (s *Service) ListWarehouses(ctx context.Context) ([]model.Warehouse, error) {
	var list []model.Warehouse
	err := s.db.WithContext(ctx).Where("status = ?", 1).Find(&list).Error
	return list, err
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
	var oc int64
	s.db.Model(&model.OpenCredential{}).Count(&oc)
	if oc == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("OpenDemo@123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := s.db.Create(&model.OpenCredential{
			ClientID: "open-demo", SecretHash: string(hash), Name: "监管入站演示凭证", Enabled: true,
		}).Error; err != nil {
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

func (s *Service) UpdateMaterial(ctx context.Context, id uint64, in CreateMaterialInput) (*model.Material, error) {
	var m model.Material
	if err := s.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%s: material", errcode.NotFound)
		}
		return nil, err
	}
	if in.Name != "" {
		m.Name = in.Name
	}
	if in.WasteCode != "" {
		m.WasteCode = in.WasteCode
	}
	if in.Category != "" {
		m.Category = in.Category
	}
	if in.Unit != "" {
		m.Unit = in.Unit
	}
	if in.Spec != "" {
		m.Spec = in.Spec
	}
	if err := s.db.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Service) ListUniqueCodes(ctx context.Context, warehouseID, status, q string, page, pageSize int) ([]model.UniqueCode, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	dbq := s.db.WithContext(ctx).Model(&model.UniqueCode{})
	if warehouseID != "" {
		dbq = dbq.Where("warehouse_id = ?", warehouseID)
	}
	if status != "" {
		dbq = dbq.Where("status = ?", status)
	}
	if q != "" {
		like := "%" + q + "%"
		dbq = dbq.Where("code LIKE ?", like)
	}
	var total int64
	_ = dbq.Count(&total)
	var list []model.UniqueCode
	err := dbq.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *Service) ValidateOpenCredential(ctx context.Context, clientID, secret string) (*model.OpenCredential, error) {
	var c model.OpenCredential
	if err := s.db.WithContext(ctx).Where("client_id = ? AND enabled = ?", clientID, true).First(&c).Error; err != nil {
		return nil, fmt.Errorf("%s: open credential", errcode.AuthDenied)
	}
	if bcrypt.CompareHashAndPassword([]byte(c.SecretHash), []byte(secret)) != nil {
		return nil, fmt.Errorf("%s: open credential", errcode.AuthDenied)
	}
	return &c, nil
}

func (s *Service) ListMaterials(ctx context.Context, page, pageSize int) ([]model.Material, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%s: inbound order", errcode.NotFound)
		}
		return nil, err
	}
	var item model.InboundOrderItem
	if err := s.db.WithContext(ctx).Where("inbound_order_id = ?", orderID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%s: inbound order item", errcode.NotFound)
		}
		return nil, err
	}
	now := time.Now()
	prefix := fmt.Sprintf("HW%s", now.Format("150405"))
	codes := make([]model.UniqueCode, 0, count)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := 0; i < count; i++ {
			c := model.UniqueCode{
				Code:           fmt.Sprintf("%s%04d%s", prefix, i+1, uuid.NewString()[:4]),
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
	if hit, err := replayed(s.db.WithContext(ctx), key); err != nil {
		return nil, err
	} else if hit {
		return &ConfirmResult{UniqueCode: in.UniqueCode, Status: model.CodeInStock, Replayed: true}, nil
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
		// 幂等键写入放在事务内：并发重复提交会命中唯一键并回滚，避免重复入账（需求 9.4）。
		if err := claimIdempotency(tx, key, code.Code); err != nil {
			return err
		}
		if err := s.enqueueBizOutbox(tx, "inbound.confirmed", "inbound", fmt.Sprintf("%d", order.ID), map[string]any{
			"orderId": order.ID, "orderNo": order.OrderNo, "warehouseId": order.WarehouseID,
			"uniqueCode": code.Code, "materialId": code.MaterialID, "operatorId": in.OperatorID,
		}); err != nil {
			return err
		}
		result = &ConfirmResult{UniqueCode: code.Code, Status: code.Status}
		return nil
	})
	if err == nil && result != nil && !result.Replayed {
		s.publish("biz.event.v1", "svc-biz", map[string]any{"eventType": "inbound.confirmed", "uniqueCode": result.UniqueCode})
	}
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
	if hit, err := replayed(s.db.WithContext(ctx), key); err != nil {
		return nil, err
	} else if hit {
		return &ConfirmResult{UniqueCode: in.UniqueCode, Status: model.CodeLocked, Replayed: true}, nil
	}

	var result *ConfirmResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.OutboundOrder
		if err := tx.First(&order, in.OrderID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%s: outbound order", errcode.NotFound)
			}
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
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		if err := claimIdempotency(tx, key, code.Code); err != nil {
			return err
		}
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
	if in.PhotoKey == "" {
		return nil, fmt.Errorf("%s: photo required for receive", errcode.ValidationError)
	}
	key := hashKey("outbound_receive:" + in.ClientRequestID)
	if hit, err := replayed(s.db.WithContext(ctx), key); err != nil {
		return nil, err
	} else if hit {
		return &ConfirmResult{UniqueCode: in.UniqueCode, Status: model.CodeSigned, Replayed: true}, nil
	}

	var result *ConfirmResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.OutboundOrder
		if err := tx.First(&order, in.OrderID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%s: outbound order", errcode.NotFound)
			}
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
		code.Status = model.CodeSigned
		code.SignedAt = &now
		if err := tx.Save(&code).Error; err != nil {
			return err
		}
		// 出库确认时已锁定库存：签收释放锁定并实扣。
		// 若该码未走出库确认（例如仅由发车打卡进入在途），锁定不存在，则直接实扣，
		// 避免 locked_qty 只增不减导致可用库存永久泄漏。
		consumed, err := tryConsumeLockedStock(tx, order.WarehouseID, code.MaterialID, 1, fmt.Sprintf("%d", order.ID), code.Code)
		if err != nil {
			return err
		}
		if !consumed {
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
		if err := claimIdempotency(tx, key, code.Code); err != nil {
			return err
		}
		// 签收后关联运输任务置为已签收（幂等：已签收的任务不再更新）。
		if err := tx.Model(&model.TransportTask{}).
			Where("outbound_order_id = ? AND status <> ?", order.ID, model.TransportSigned).
			Update("status", model.TransportSigned).Error; err != nil {
			return err
		}
		if err := s.enqueueBizOutbox(tx, "outbound.received", "outbound", fmt.Sprintf("%d", order.ID), map[string]any{
			"orderId": order.ID, "warehouseId": order.WarehouseID,
			"uniqueCode": code.Code, "materialId": code.MaterialID, "operatorId": in.OperatorID,
		}); err != nil {
			return err
		}
		result = &ConfirmResult{UniqueCode: code.Code, Status: code.Status}
		return nil
	})
	if err == nil && result != nil && !result.Replayed {
		s.publish("biz.event.v1", "svc-biz", map[string]any{"eventType": "outbound.received", "uniqueCode": result.UniqueCode})
	}
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

// VoidUniqueCode transitions a code to voided. Signed codes are rejected by the state machine.
// In-stock / locked codes also adjust stock balances so book qty stays consistent.
func (s *Service) VoidUniqueCode(ctx context.Context, code string, actorID uint64, reason string) (*model.UniqueCode, error) {
	var out *model.UniqueCode
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var c model.UniqueCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", code).First(&c).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%s: unique code", errcode.NotFound)
			}
			return err
		}
		if c.Status == model.CodeSigned {
			return fmt.Errorf("%s: signed cannot void", errcode.StateInvalid)
		}
		if err := domain.CanTransit(c.Status, model.CodeVoided); err != nil {
			return err
		}
		switch c.Status {
		case model.CodeInStock:
			if err := adjustStock(tx, c.WarehouseID, c.MaterialID, -1, "code_void", code, code); err != nil {
				return err
			}
		case model.CodeLocked:
			ok, err := tryConsumeLockedStock(tx, c.WarehouseID, c.MaterialID, 1, "code_void:"+code, code)
			if err != nil {
				return err
			}
			if !ok {
				// lock book missing: best-effort qty decrement, then still void
				_ = adjustStock(tx, c.WarehouseID, c.MaterialID, -1, "code_void", code, code)
			}
		}
		c.Status = model.CodeVoided
		if err := tx.Save(&c).Error; err != nil {
			return err
		}
		_ = reason
		_ = actorID
		out = &c
		return nil
	})
	return out, err
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

func (s *Service) ListOutboundOrders(ctx context.Context, page, pageSize int) ([]model.OutboundOrder, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	var total int64
	q := s.db.WithContext(ctx).Model(&model.OutboundOrder{})
	_ = q.Count(&total)
	var list []model.OutboundOrder
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *Service) ListInboundOrders(ctx context.Context, page, pageSize int) ([]model.InboundOrder, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
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

// consumeLockedStock 释放锁定并实扣；锁定不足时返回 STATE_INVALID 且不修改任何数据。
func consumeLockedStock(tx *gorm.DB, warehouseID string, materialID uint64, qty float64, bizID, uniqueCode string) error {
	ok, err := tryConsumeLockedStock(tx, warehouseID, materialID, qty, bizID, uniqueCode)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s: insufficient locked stock", errcode.StateInvalid)
	}
	return nil
}

// tryConsumeLockedStock 尝试释放锁定并实扣；返回 false 表示无足够锁定量（未做修改）。
func tryConsumeLockedStock(tx *gorm.DB, warehouseID string, materialID uint64, qty float64, bizID, uniqueCode string) (bool, error) {
	var bal model.StockBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("warehouse_id = ? AND material_id = ?", warehouseID, materialID).
		First(&bal).Error; err != nil {
		return false, err
	}
	if bal.LockedQty < qty || bal.Qty < qty {
		return false, nil
	}
	bal.LockedQty -= qty
	bal.Qty -= qty
	if err := tx.Save(&bal).Error; err != nil {
		return false, err
	}
	return true, tx.Create(&model.StockLedger{
		WarehouseID: warehouseID, MaterialID: materialID, Delta: -qty,
		BalanceAfter: bal.Qty, BizType: "outbound_receive", BizID: bizID, UniqueCode: uniqueCode,
	}).Error
}

// normalizePage 统一分页参数：page 从 1 起，pageSize 默认 20、最大 100（需求 1.3）。
func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

// replayed 判断幂等键是否已处理过；查询出错（非「记录不存在」）时向上抛出，避免把故障当新请求执行。
func replayed(db *gorm.DB, key string) (bool, error) {
	var rec model.IdempotencyRecord
	err := db.Where("key_hash = ?", key).First(&rec).Error
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return false, nil
	default:
		return false, err
	}
}

// claimIdempotency 在事务内写入幂等记录；命中唯一键说明是重放，返回 IDEMPOTENT_REPLAY 让事务回滚。
func claimIdempotency(tx *gorm.DB, key, response string) error {
	err := tx.Create(&model.IdempotencyRecord{KeyHash: key, Response: response}).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return fmt.Errorf("%s: duplicate clientRequestId", errcode.IdempotentReplay)
	}
	if err != nil && strings.Contains(err.Error(), "1062") { // MySQL 唯一键冲突兜底
		return fmt.Errorf("%s: duplicate clientRequestId", errcode.IdempotentReplay)
	}
	return err
}

func hashKey(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
