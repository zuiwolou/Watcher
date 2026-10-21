package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"watcher/backend/pkg/auth"
	"watcher/backend/pkg/authz"
	"watcher/backend/pkg/errcode"
	"watcher/backend/pkg/ldapx"
	"watcher/backend/pkg/response"
	"watcher/backend/internal/webhttp/iam"
	"watcher/backend/internal/webhttp/middleware"
)

type AuthHandler struct {
	db     *gorm.DB
	jwt    *auth.Manager
	e      *casbin.Enforcer
	ldap   *ldapx.Client
	defWH  string
}

func NewAuthHandler(db *gorm.DB, jwt *auth.Manager, e *casbin.Enforcer, ldap *ldapx.Client, defaultWH string) *AuthHandler {
	return &AuthHandler{db: db, jwt: jwt, e: e, ldap: ldap, defWH: defaultWH}
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Mode     string `json:"mode"` // local | ldap
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	mode := req.Mode
	if mode == "" {
		mode = "local"
	}

	var user iam.User
	switch mode {
	case "ldap":
		if h.ldap == nil || !h.ldap.Enabled() {
			response.Fail(c, http.StatusServiceUnavailable, errcode.LDAPError, "ldap not configured")
			return
		}
		dn, err := h.ldap.Authenticate(req.Username, req.Password)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, errcode.LDAPError, "ldap auth failed")
			return
		}
		err = h.db.Where("username = ?", req.Username).First(&user).Error
		if err == gorm.ErrRecordNotFound {
			user = iam.User{
				Username: req.Username, DisplayName: req.Username, Status: 1,
				LDAPDN: dn, MustChangePassword: false,
			}
			if err := h.db.Create(&user).Error; err != nil {
				response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
				return
			}
			_ = h.db.Create(&iam.UserClientAccess{UserID: user.ID, Client: "portal"})
			_ = h.db.Create(&iam.UserClientAccess{UserID: user.ID, Client: "app"})
			if h.defWH != "" {
				_ = h.db.Create(&iam.UserScope{UserID: user.ID, WarehouseID: h.defWH})
			}
			// default readonly role if exists
			var role iam.Role
			if h.db.Where("code = ?", "readonly").First(&role).Error == nil {
				_ = h.db.Create(&iam.UserRole{UserID: user.ID, RoleID: role.ID})
				_, _ = authz.AddRoleForUser(h.e, strconv.FormatUint(user.ID, 10), "readonly")
				_ = h.e.SavePolicy()
			}
		} else if err != nil {
			response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
			return
		}
		if user.Status != 1 {
			response.Fail(c, http.StatusUnauthorized, errcode.AuthDenied, "user disabled")
			return
		}
		user.LDAPDN = dn
		_ = h.db.Save(&user)
	default:
		if err := h.db.Where("username = ? AND status = 1", req.Username).First(&user).Error; err != nil {
			response.Fail(c, http.StatusUnauthorized, errcode.AuthDenied, "invalid username or password")
			return
		}
		if user.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
			response.Fail(c, http.StatusUnauthorized, errcode.AuthDenied, "invalid username or password")
			return
		}
	}

	h.respondTokens(c, &user)
}

func (h *AuthHandler) respondTokens(c *gin.Context, user *iam.User) {
	perms, err := iam.ListPermissions(h.e, user.ID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
		return
	}
	var scopes []iam.UserScope
	_ = h.db.Where("user_id = ?", user.ID).Find(&scopes)
	whs := make([]string, 0, len(scopes))
	for _, s := range scopes {
		whs = append(whs, s.WarehouseID)
	}
	var clients []iam.UserClientAccess
	_ = h.db.Where("user_id = ?", user.ID).Find(&clients)
	cs := make([]string, 0, len(clients))
	for _, x := range clients {
		cs = append(cs, x.Client)
	}

	uid := strconv.FormatUint(user.ID, 10)
	access, expiresIn, err := h.jwt.IssueAccess(uid, user.Username, perms, whs, cs)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
		return
	}
	refreshPlain := uuid.NewString()
	sum := sha256.Sum256([]byte(refreshPlain))
	rt := iam.RefreshToken{
		UserID: user.ID, TokenHash: hex.EncodeToString(sum[:]),
		ExpiresAt: time.Now().Add(h.jwt.RefreshTTL()),
	}
	if err := h.db.Create(&rt).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
		return
	}

	response.OK(c, gin.H{
		"accessToken": access, "refreshToken": refreshPlain, "expiresIn": expiresIn,
		"permissions": perms, "scopes": gin.H{"warehouseIds": whs}, "clients": cs,
		"pushEndpoint": "/v1/stream", "mustChangePassword": user.MustChangePassword,
		"user": gin.H{"id": uid, "displayName": user.DisplayName, "username": user.Username},
	})
}

type refreshReq struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	sum := sha256.Sum256([]byte(req.RefreshToken))
	hash := hex.EncodeToString(sum[:])
	var rt iam.RefreshToken
	if err := h.db.Where("token_hash = ? AND revoked = ? AND expires_at > ?", hash, false, time.Now()).First(&rt).Error; err != nil {
		response.Fail(c, http.StatusUnauthorized, errcode.AuthDenied, "invalid refresh token")
		return
	}
	var user iam.User
	if err := h.db.First(&user, rt.UserID).Error; err != nil || user.Status != 1 {
		response.Fail(c, http.StatusUnauthorized, errcode.AuthDenied, "user disabled")
		return
	}
	rt.Revoked = true
	_ = h.db.Save(&rt)

	perms, _ := iam.ListPermissions(h.e, user.ID)
	var scopes []iam.UserScope
	_ = h.db.Where("user_id = ?", user.ID).Find(&scopes)
	whs := make([]string, 0, len(scopes))
	for _, s := range scopes {
		whs = append(whs, s.WarehouseID)
	}
	var clients []iam.UserClientAccess
	_ = h.db.Where("user_id = ?", user.ID).Find(&clients)
	cs := make([]string, 0, len(clients))
	for _, x := range clients {
		cs = append(cs, x.Client)
	}
	uid := strconv.FormatUint(user.ID, 10)
	access, expiresIn, err := h.jwt.IssueAccess(uid, user.Username, perms, whs, cs)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
		return
	}
	newPlain := uuid.NewString()
	sum2 := sha256.Sum256([]byte(newPlain))
	_ = h.db.Create(&iam.RefreshToken{
		UserID: user.ID, TokenHash: hex.EncodeToString(sum2[:]),
		ExpiresAt: time.Now().Add(h.jwt.RefreshTTL()),
	}).Error

	response.OK(c, gin.H{"accessToken": access, "refreshToken": newPlain, "expiresIn": expiresIn})
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid, _ := c.Get(middleware.CtxUserID)
	response.OK(c, gin.H{
		"id": uid, "username": c.GetString(middleware.CtxUsername),
		"permissions": c.MustGet(middleware.CtxPermissions),
		"warehouseIds": c.MustGet(middleware.CtxWarehouseIDs),
		"clients": c.MustGet(middleware.CtxClients),
	})
}
