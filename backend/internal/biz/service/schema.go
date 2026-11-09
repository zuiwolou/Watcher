package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"watcher/backend/internal/biz/internal/model"
	"watcher/backend/pkg/bus"
	"watcher/backend/pkg/errcode"
	"watcher/backend/pkg/mqttmap"
)

func (s *Service) ListMqttSchemas(ctx context.Context) ([]model.MqttSchemaProfile, error) {
	var list []model.MqttSchemaProfile
	err := s.db.WithContext(ctx).Order("id desc").Find(&list).Error
	return list, err
}

func (s *Service) UpsertMqttSchema(ctx context.Context, id uint64, code, name, topic, mapping string, enabled bool) (*model.MqttSchemaProfile, error) {
	if code == "" {
		return nil, fmt.Errorf("%s: code required", errcode.ValidationError)
	}
	if mapping != "" {
		if _, err := mqttmap.ParseMapping(mapping); err != nil {
			return nil, fmt.Errorf("%s: %v", errcode.ValidationError, err)
		}
	}
	var out *model.MqttSchemaProfile
	if id > 0 {
		var p model.MqttSchemaProfile
		if err := s.db.First(&p, id).Error; err != nil {
			return nil, err
		}
		p.Code, p.Name, p.TopicPattern, p.MappingJSON, p.Enabled = code, name, topic, mapping, enabled
		if err := s.db.Save(&p).Error; err != nil {
			return nil, err
		}
		out = &p
	} else {
		p := &model.MqttSchemaProfile{Code: code, Name: name, TopicPattern: topic, MappingJSON: mapping, Enabled: enabled}
		if err := s.db.Create(p).Error; err != nil {
			return nil, err
		}
		out = p
	}
	s.publish(bus.TopicConfigMQTT, "svc-webhttp", map[string]any{
		"action": "upsert", "id": out.ID, "code": out.Code, "topicPattern": out.TopicPattern, "enabled": out.Enabled,
	})
	return out, nil
}

// DryRunMqttSchema validates mapping JSON and optionally applies it to a sample payload.
func (s *Service) DryRunMqttSchema(ctx context.Context, mappingJSON, samplePayload string) (map[string]any, error) {
	_ = ctx
	mp, err := mqttmap.ParseMapping(mappingJSON)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", errcode.ValidationError, err)
	}
	out := map[string]any{"ok": true, "mapping": mp}
	if samplePayload != "" {
		msg, err := mqttmap.Apply(mappingJSON, []byte(samplePayload))
		if err != nil {
			return nil, fmt.Errorf("%s: sample apply: %v", errcode.ValidationError, err)
		}
		out["canonical"] = msg
	}
	return out, nil
}

func (s *Service) ListIntegrations(ctx context.Context) ([]model.IntegrationConfig, error) {
	var list []model.IntegrationConfig
	err := s.db.WithContext(ctx).Find(&list).Error
	return list, err
}

func (s *Service) UpsertIntegration(ctx context.Context, id uint64, code, name, cfgJSON string, enabled bool) (*model.IntegrationConfig, error) {
	if code == "" {
		return nil, fmt.Errorf("%s: code required", errcode.ValidationError)
	}
	if cfgJSON != "" {
		var tmp any
		if err := json.Unmarshal([]byte(cfgJSON), &tmp); err != nil {
			return nil, fmt.Errorf("%s: configJson invalid", errcode.ValidationError)
		}
	}
	if id > 0 {
		var p model.IntegrationConfig
		if err := s.db.First(&p, id).Error; err != nil {
			return nil, err
		}
		p.Code, p.Name, p.ConfigJSON, p.Enabled = code, name, cfgJSON, enabled
		if err := s.db.Save(&p).Error; err != nil {
			return nil, err
		}
		return &p, nil
	}
	var existing model.IntegrationConfig
	err := s.db.Where("code = ?", code).First(&existing).Error
	if err == nil {
		existing.Name, existing.ConfigJSON, existing.Enabled = name, cfgJSON, enabled
		if err := s.db.Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	p := &model.IntegrationConfig{Code: code, Name: name, ConfigJSON: cfgJSON, Enabled: enabled}
	if err := s.db.Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}
