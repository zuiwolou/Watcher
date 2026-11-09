package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"watcher/backend/internal/biz/internal/model"
	"watcher/backend/pkg/errcode"
)

type DeviceInput struct {
	DeviceCode      string
	Name            string
	Type            string
	WarehouseID     string
	SchemaProfileID *uint64
	AuthType        string
	Status          string
	MetaJSON        string
}

func (s *Service) ListDevices(ctx context.Context, warehouseID string) ([]model.Device, error) {
	var list []model.Device
	q := s.db.WithContext(ctx).Model(&model.Device{})
	if warehouseID != "" {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	err := q.Order("id desc").Find(&list).Error
	return list, err
}

func (s *Service) GetDevice(ctx context.Context, id uint64) (*model.Device, error) {
	var d model.Device
	if err := s.db.WithContext(ctx).First(&d, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%s: device", errcode.NotFound)
		}
		return nil, err
	}
	return &d, nil
}

func (s *Service) CreateDevice(ctx context.Context, in DeviceInput) (*model.Device, error) {
	if in.DeviceCode == "" {
		return nil, fmt.Errorf("%s: deviceCode required", errcode.ValidationError)
	}
	if in.Type == "" {
		in.Type = "warehouse"
	}
	if in.AuthType == "" {
		in.AuthType = "password"
	}
	if in.Status == "" {
		in.Status = model.DeviceActive
	}
	if in.MetaJSON != "" {
		var tmp any
		if err := json.Unmarshal([]byte(in.MetaJSON), &tmp); err != nil {
			return nil, fmt.Errorf("%s: metaJson invalid", errcode.ValidationError)
		}
	}
	if in.SchemaProfileID != nil && *in.SchemaProfileID > 0 {
		var n int64
		s.db.WithContext(ctx).Model(&model.MqttSchemaProfile{}).Where("id = ?", *in.SchemaProfileID).Count(&n)
		if n == 0 {
			return nil, fmt.Errorf("%s: schemaProfileId", errcode.NotFound)
		}
	}
	d := &model.Device{
		DeviceCode: in.DeviceCode, Name: in.Name, Type: in.Type, WarehouseID: in.WarehouseID,
		SchemaProfileID: in.SchemaProfileID, AuthType: in.AuthType, Status: in.Status, MetaJSON: in.MetaJSON,
	}
	if err := s.db.WithContext(ctx).Create(d).Error; err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) UpdateDevice(ctx context.Context, id uint64, in DeviceInput) (*model.Device, error) {
	d, err := s.GetDevice(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != "" {
		d.Name = in.Name
	}
	if in.Type != "" {
		d.Type = in.Type
	}
	if in.WarehouseID != "" {
		d.WarehouseID = in.WarehouseID
	}
	if in.AuthType != "" {
		d.AuthType = in.AuthType
	}
	if in.Status != "" {
		d.Status = in.Status
	}
	if in.MetaJSON != "" {
		var tmp any
		if err := json.Unmarshal([]byte(in.MetaJSON), &tmp); err != nil {
			return nil, fmt.Errorf("%s: metaJson invalid", errcode.ValidationError)
		}
		d.MetaJSON = in.MetaJSON
	}
	if in.SchemaProfileID != nil {
		if *in.SchemaProfileID > 0 {
			var n int64
			s.db.WithContext(ctx).Model(&model.MqttSchemaProfile{}).Where("id = ?", *in.SchemaProfileID).Count(&n)
			if n == 0 {
				return nil, fmt.Errorf("%s: schemaProfileId", errcode.NotFound)
			}
		}
		d.SchemaProfileID = in.SchemaProfileID
	}
	if err := s.db.WithContext(ctx).Save(d).Error; err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) BindDeviceMqttSchema(ctx context.Context, id, schemaProfileID uint64) (*model.Device, error) {
	d, err := s.GetDevice(ctx, id)
	if err != nil {
		return nil, err
	}
	if schemaProfileID > 0 {
		var n int64
		s.db.WithContext(ctx).Model(&model.MqttSchemaProfile{}).Where("id = ?", schemaProfileID).Count(&n)
		if n == 0 {
			return nil, fmt.Errorf("%s: schemaProfileId", errcode.NotFound)
		}
		d.SchemaProfileID = &schemaProfileID
	} else {
		d.SchemaProfileID = nil
	}
	if err := s.db.WithContext(ctx).Save(d).Error; err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) FindDeviceByCode(ctx context.Context, code string) (*model.Device, error) {
	var d model.Device
	if err := s.db.WithContext(ctx).Where("device_code = ?", code).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}
