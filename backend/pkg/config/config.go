package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Env      string
	HTTPAddr string
	GRPCAddr string
	BizGRPC  string

	MySQLDSN   string
	IAMDSN     string
	BizDSN     string
	HistoryDSN string

	JWTSecret          string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	CasbinModelPath    string
	SeedAdminUser      string
	SeedAdminPassword  string
	DefaultWarehouseID string

	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool
	MinIOPublicURL string

	LDAPEnabled  bool
	LDAPURL      string
	LDAPBindDN   string
	LDAPBindPass string
	LDAPBaseDN   string
	LDAPFilter   string
	LDAPStartTLS bool
}

func Load(service string) (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	v.AddConfigPath("./backend/configs")
	v.AddConfigPath("/etc/watcher")

	v.SetEnvPrefix("WATCHER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	_ = v.ReadInConfig()

	setDefaults(v)

	cfg := &Config{
		Env:                v.GetString("env"),
		HTTPAddr:           v.GetString("http.addr"),
		GRPCAddr:           v.GetString("grpc.addr"),
		BizGRPC:            v.GetString("biz.grpc"),
		MySQLDSN:           v.GetString("mysql.dsn"),
		IAMDSN:             firstNonEmpty(v.GetString("mysql.iam_dsn"), v.GetString("mysql.dsn")),
		BizDSN:             firstNonEmpty(v.GetString("mysql.biz_dsn"), v.GetString("mysql.dsn")),
		HistoryDSN:         firstNonEmpty(v.GetString("mysql.history_dsn"), v.GetString("mysql.dsn")),
		JWTSecret:          v.GetString("jwt.secret"),
		AccessTokenTTL:     v.GetDuration("jwt.access_ttl"),
		RefreshTokenTTL:    v.GetDuration("jwt.refresh_ttl"),
		CasbinModelPath:    v.GetString("casbin.model"),
		SeedAdminUser:      v.GetString("seed.admin_user"),
		SeedAdminPassword:  v.GetString("seed.admin_password"),
		DefaultWarehouseID: v.GetString("seed.warehouse_id"),
		MinIOEndpoint:      v.GetString("minio.endpoint"),
		MinIOAccessKey:     v.GetString("minio.access_key"),
		MinIOSecretKey:     v.GetString("minio.secret_key"),
		MinIOBucket:        v.GetString("minio.bucket"),
		MinIOUseSSL:        v.GetBool("minio.use_ssl"),
		MinIOPublicURL:     v.GetString("minio.public_url"),
		LDAPEnabled:        v.GetBool("ldap.enabled"),
		LDAPURL:            v.GetString("ldap.url"),
		LDAPBindDN:         v.GetString("ldap.bind_dn"),
		LDAPBindPass:       v.GetString("ldap.bind_password"),
		LDAPBaseDN:         v.GetString("ldap.base_dn"),
		LDAPFilter:         v.GetString("ldap.user_filter"),
		LDAPStartTLS:       v.GetBool("ldap.start_tls"),
	}
	_ = service
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("env", "dev")
	v.SetDefault("http.addr", ":8080")
	v.SetDefault("grpc.addr", ":9091")
	v.SetDefault("biz.grpc", "127.0.0.1:9091")
	v.SetDefault("mysql.dsn", "watcher:watcher@tcp(127.0.0.1:3306)/watcher?charset=utf8mb4&parseTime=True&loc=Local")
	v.SetDefault("jwt.secret", "dev-change-me-watcher-secret")
	v.SetDefault("jwt.access_ttl", 30*time.Minute)
	v.SetDefault("jwt.refresh_ttl", 7*24*time.Hour)
	v.SetDefault("casbin.model", "backend/configs/casbin_model.conf")
	v.SetDefault("seed.admin_user", "admin")
	v.SetDefault("seed.admin_password", "Admin@123")
	v.SetDefault("seed.warehouse_id", "WH01")
	v.SetDefault("minio.endpoint", "127.0.0.1:9000")
	v.SetDefault("minio.access_key", "minioadmin")
	v.SetDefault("minio.secret_key", "minioadmin")
	v.SetDefault("minio.bucket", "watcher")
	v.SetDefault("minio.use_ssl", false)
	v.SetDefault("minio.public_url", "http://127.0.0.1:9000")
	v.SetDefault("ldap.enabled", false)
	v.SetDefault("ldap.user_filter", "(uid=%s)")
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
