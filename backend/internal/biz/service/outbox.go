package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"watcher/backend/internal/biz/internal/model"
	"watcher/backend/pkg/errcode"
)

// EnqueueOutbox writes a pending outbox row inside an existing transaction.
func EnqueueOutbox(tx *gorm.DB, channel, eventType, bizType, bizID string, payload any) error {
	if tx == nil {
		return fmt.Errorf("%s: tx required", errcode.ValidationError)
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	now := time.Now()
	row := &model.IntegrationOutbox{
		Channel: channel, EventType: eventType, BizType: bizType, BizID: bizID,
		PayloadJSON: string(b), Status: model.OutboxPending, Attempts: 0, MaxAttempts: 8,
		NextRetryAt: &now,
	}
	return tx.Create(row).Error
}

func (s *Service) enqueueBizOutbox(tx *gorm.DB, eventType, bizType, bizID string, payload any) error {
	// fan-out to every enabled integration channel that matches code prefix
	var cfgs []model.IntegrationConfig
	if err := tx.Where("enabled = ?", true).Find(&cfgs).Error; err != nil {
		return err
	}
	if len(cfgs) == 0 {
		// still enqueue openapi placeholder so ops can inspect when config enabled later
		return EnqueueOutbox(tx, "openapi", eventType, bizType, bizID, payload)
	}
	for _, c := range cfgs {
		ch := c.Code
		if ch != "openapi" && ch != "sftp" {
			ch = "openapi"
		}
		if err := EnqueueOutbox(tx, ch, eventType, bizType, bizID, payload); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ListOutbox(ctx context.Context, page, pageSize int, status, channel string) ([]model.IntegrationOutbox, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	q := s.db.WithContext(ctx).Model(&model.IntegrationOutbox{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if channel != "" {
		q = q.Where("channel = ?", channel)
	}
	var total int64
	_ = q.Count(&total)
	var list []model.IntegrationOutbox
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *Service) RetryOutbox(ctx context.Context, id uint64) (*model.IntegrationOutbox, error) {
	var row model.IntegrationOutbox
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, fmt.Errorf("%s: outbox", errcode.NotFound)
	}
	now := time.Now()
	row.Status = model.OutboxPending
	row.NextRetryAt = &now
	row.LastError = ""
	if err := s.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

type outboxEndpoint struct {
	Endpoint   string            `json:"endpoint"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers"`
	TimeoutSec int               `json:"timeoutSec"`
	Path       string            `json:"path"` // sftp remote dir or local drop dir
	Host       string            `json:"host"`
	Port       int               `json:"port"`
	Username   string            `json:"username"`
	Password   string            `json:"password"`
	// FieldMap maps internal payload keys -> regulator keys (BE-08).
	// Example: {"alarmId":"alarm_id","warehouseId":"wh_code"}
	FieldMap map[string]string `json:"fieldMap"`
}

// applyFieldMap renames top-level payload keys when FieldMap is set; unmapped keys kept.
func applyFieldMap(payloadJSON string, fieldMap map[string]string) ([]byte, error) {
	var body map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &body); err != nil {
		return []byte(payloadJSON), nil
	}
	if len(fieldMap) == 0 {
		return json.Marshal(body)
	}
	out := make(map[string]any, len(body))
	for k, v := range body {
		if nk, ok := fieldMap[k]; ok && nk != "" {
			out[nk] = v
		} else {
			out[k] = v
		}
	}
	return json.Marshal(out)
}

// ProcessOutboxBatch delivers pending rows (true HTTP for openapi; sftp writes local drop file).
func (s *Service) ProcessOutboxBatch(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 20
	}
	now := time.Now()
	var rows []model.IntegrationOutbox
	err := s.db.WithContext(ctx).
		Where("status IN ? AND (next_retry_at IS NULL OR next_retry_at <= ?)",
			[]string{model.OutboxPending, model.OutboxFailed}, now).
		Order("id asc").Limit(limit).Find(&rows).Error
	if err != nil {
		return 0, err
	}
	done := 0
	for i := range rows {
		if err := s.deliverOutbox(ctx, &rows[i]); err == nil {
			done++
		}
	}
	return done, nil
}

func (s *Service) deliverOutbox(ctx context.Context, row *model.IntegrationOutbox) error {
	// claim
	res := s.db.WithContext(ctx).Model(&model.IntegrationOutbox{}).
		Where("id = ? AND status IN ?", row.ID, []string{model.OutboxPending, model.OutboxFailed}).
		Updates(map[string]any{"status": model.OutboxDelivering, "attempts": gorm.Expr("attempts + 1")})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil
	}
	_ = s.db.WithContext(ctx).First(row, row.ID)

	var cfg model.IntegrationConfig
	err := s.db.WithContext(ctx).Where("code = ? AND enabled = ?", row.Channel, true).First(&cfg).Error
	if err != nil {
		return s.failOutbox(row, fmt.Errorf("integration %s not enabled: %w", row.Channel, err))
	}
	var ep outboxEndpoint
	_ = json.Unmarshal([]byte(cfg.ConfigJSON), &ep)

	var deliverErr error
	switch row.Channel {
	case "sftp":
		deliverErr = deliverSFTPDrop(ep, row)
	default:
		deliverErr = deliverHTTP(ctx, ep, row)
	}
	if deliverErr != nil {
		return s.failOutbox(row, deliverErr)
	}
	now := time.Now()
	row.Status = model.OutboxSuccess
	row.DeliveredAt = &now
	row.LastError = ""
	return s.db.Save(row).Error
}

func deliverHTTP(ctx context.Context, ep outboxEndpoint, row *model.IntegrationOutbox) error {
	if ep.Endpoint == "" {
		return fmt.Errorf("openapi endpoint empty")
	}
	method := ep.Method
	if method == "" {
		method = http.MethodPost
	}
	timeout := time.Duration(ep.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	mapped, err := applyFieldMap(row.PayloadJSON, ep.FieldMap)
	if err != nil {
		return err
	}
	body := map[string]any{}
	_ = json.Unmarshal(mapped, &body)
	envelope := map[string]any{
		"eventType":  row.EventType,
		"bizType":    row.BizType,
		"bizId":      row.BizID,
		"channel":    row.Channel,
		"outboxId":   row.ID,
		"payload":    body,
		"occurredAt": time.Now().UTC().Format(time.RFC3339),
	}
	b, _ := json.Marshal(envelope)
	req, err := http.NewRequestWithContext(ctx, method, ep.Endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range ep.Headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	return nil
}

func deliverSFTPDrop(ep outboxEndpoint, row *model.IntegrationOutbox) error {
	mapped, err := applyFieldMap(row.PayloadJSON, ep.FieldMap)
	if err != nil {
		return err
	}
	fileName := fmt.Sprintf("%d_%s_%s.json", row.ID, row.EventType, row.BizID)
	// Host 为空：本地落盘（开发/演示）；配置 Host 后走真 SFTP。
	if strings.TrimSpace(ep.Host) == "" {
		dir := ep.Path
		if dir == "" {
			dir = "tmp/sftp-drop"
		}
		name := filepath.Join(strings.TrimRight(dir, `/\`), fileName)
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			return err
		}
		return os.WriteFile(name, mapped, 0o644)
	}
	return deliverSFTPRemote(ep, fileName, mapped)
}

func (s *Service) failOutbox(row *model.IntegrationOutbox, cause error) error {
	row.LastError = cause.Error()
	if len(row.LastError) > 500 {
		row.LastError = row.LastError[:500]
	}
	if row.Attempts >= row.MaxAttempts {
		row.Status = model.OutboxDead
		row.NextRetryAt = nil
	} else {
		row.Status = model.OutboxFailed
		backoff := time.Duration(1<<minInt(row.Attempts, 6)) * time.Second
		t := time.Now().Add(backoff)
		row.NextRetryAt = &t
	}
	_ = s.db.Save(row).Error
	return cause
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

