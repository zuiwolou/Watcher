package ldapx

import (
	"crypto/tls"
	"fmt"

	"github.com/go-ldap/ldap/v3"
)

type Config struct {
	Enabled    bool
	URL        string // ldap://host:389 or ldaps://
	BindDN     string
	BindPass   string
	BaseDN     string
	UserFilter string // e.g. (uid=%s) or (sAMAccountName=%s)
	StartTLS   bool
}

type Client struct {
	cfg Config
}

func New(cfg Config) *Client {
	if cfg.UserFilter == "" {
		cfg.UserFilter = "(uid=%s)"
	}
	return &Client{cfg: cfg}
}

func (c *Client) Enabled() bool { return c.cfg.Enabled && c.cfg.URL != "" && c.cfg.BaseDN != "" }

// Authenticate binds as service account, searches user, then bind-as-user to verify password.
// Returns user DN on success.
func (c *Client) Authenticate(username, password string) (dn string, err error) {
	if !c.Enabled() {
		return "", fmt.Errorf("ldap disabled")
	}
	conn, err := ldap.DialURL(c.cfg.URL)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if c.cfg.StartTLS {
		if err := conn.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil { //nolint:gosec
			return "", err
		}
	}
	if c.cfg.BindDN != "" {
		if err := conn.Bind(c.cfg.BindDN, c.cfg.BindPass); err != nil {
			return "", fmt.Errorf("ldap bind: %w", err)
		}
	}
	filter := fmt.Sprintf(c.cfg.UserFilter, ldap.EscapeFilter(username))
	req := ldap.NewSearchRequest(
		c.cfg.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 0, false,
		filter, []string{"dn"}, nil,
	)
	res, err := conn.Search(req)
	if err != nil {
		return "", err
	}
	if len(res.Entries) != 1 {
		return "", fmt.Errorf("user not found")
	}
	dn = res.Entries[0].DN
	if err := conn.Bind(dn, password); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}
	return dn, nil
}
