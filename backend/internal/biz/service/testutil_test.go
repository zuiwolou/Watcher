package service

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	s := New(db)
	if err := s.AutoMigrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := s.SeedWarehouse("WH01", "WH01", "测试仓"); err != nil {
		t.Fatalf("seed wh: %v", err)
	}
	if err := s.SeedDefaults(); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}
	return db
}
