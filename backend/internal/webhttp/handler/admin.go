package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"watcher/backend/pkg/authz"
	"watcher/backend/pkg/errcode"
	"watcher/backend/pkg/response"
	"watcher/backend/internal/webhttp/iam"
	"watcher/backend/internal/webhttp/middleware"
)

type AdminHandler struct {
	db *gorm.DB
	e  *casbin.Enforcer
}

func NewAdminHandler(db *gorm.DB, e *casbin.Enforcer) *AdminHandler {
	return &AdminHandler{db: db, e: e}
}

func (h *AdminHandler) ListPermissions(c *gin.Context) {
	response.OK(c, gin.H{"list": iam.CatalogStrings()})
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	var total int64
	q := h.db.Model(&iam.User{})
	_ = q.Count(&total)
	var users []iam.User
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&users).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
		return
	}
	type row struct {
		iam.User
		PasswordHash string   `json:"-"`
		RoleCodes    []string `json:"roleCodes"`
		Clients      []string `json:"clients"`
		WarehouseIDs []string `json:"warehouseIds"`
	}
	out := make([]row, 0, len(users))
	for _, u := range users {
		r := row{User: u}
		r.PasswordHash = ""
		var urs []iam.UserRole
		_ = h.db.Where("user_id = ?", u.ID).Find(&urs)
		for _, ur := range urs {
			var role iam.Role
			if h.db.First(&role, ur.RoleID).Error == nil {
				r.RoleCodes = append(r.RoleCodes, role.Code)
			}
		}
		var cas []iam.UserClientAccess
		_ = h.db.Where("user_id = ?", u.ID).Find(&cas)
		for _, x := range cas {
			r.Clients = append(r.Clients, x.Client)
		}
		var sc []iam.UserScope
		_ = h.db.Where("user_id = ?", u.ID).Find(&sc)
		for _, x := range sc {
			r.WarehouseIDs = append(r.WarehouseIDs, x.WarehouseID)
		}
		out = append(out, r)
	}
	response.Page(c, out, page, size, total)
}

type userWriteReq struct {
	Username           string   `json:"username"`
	Password           string   `json:"password"`
	DisplayName        string   `json:"displayName"`
	Mobile             string   `json:"mobile"`
	Dept               string   `json:"dept"`
	Status             *int8    `json:"status"`
	MustChangePassword *bool    `json:"mustChangePassword"`
	RoleCodes          []string `json:"roleCodes"`
	Clients            []string `json:"clients"`
	WarehouseIDs       []string `json:"warehouseIds"`
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req userWriteReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, "username and password required")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
		return
	}
	must := true
	if req.MustChangePassword != nil {
		must = *req.MustChangePassword
	}
	u := iam.User{
		Username:           req.Username,
		PasswordHash:       string(hash),
		DisplayName:        req.DisplayName,
		Mobile:             req.Mobile,
		Dept:               req.Dept,
		Status:             1,
		MustChangePassword: must,
	}
	if req.Status != nil {
		u.Status = *req.Status
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&u).Error; err != nil {
			return err
		}
		return h.syncUserRelations(tx, u.ID, req.RoleCodes, req.Clients, req.WarehouseIDs)
	}); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	writeAudit(c, h.db, "user.create", "user", fmt.Sprintf("%d", u.ID), gin.H{"username": u.Username})
	response.OK(c, gin.H{"id": fmt.Sprintf("%d", u.ID)})
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var u iam.User
	if err := h.db.First(&u, id).Error; err != nil {
		response.Fail(c, http.StatusNotFound, errcode.NotFound, "user not found")
		return
	}
	var req userWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	if req.DisplayName != "" {
		u.DisplayName = req.DisplayName
	}
	if req.Mobile != "" {
		u.Mobile = req.Mobile
	}
	if req.Dept != "" {
		u.Dept = req.Dept
	}
	if req.Status != nil {
		u.Status = *req.Status
	}
	if req.MustChangePassword != nil {
		u.MustChangePassword = *req.MustChangePassword
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
			return
		}
		u.PasswordHash = string(hash)
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&u).Error; err != nil {
			return err
		}
		if req.RoleCodes != nil || req.Clients != nil || req.WarehouseIDs != nil {
			return h.syncUserRelations(tx, u.ID, req.RoleCodes, req.Clients, req.WarehouseIDs)
		}
		return nil
	}); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	// revoke refresh tokens when disabled
	if u.Status == 0 {
		_ = h.db.Model(&iam.RefreshToken{}).Where("user_id = ?", u.ID).Update("revoked", true)
	}
	writeAudit(c, h.db, "user.update", "user", fmt.Sprintf("%d", u.ID), gin.H{"status": u.Status, "roleCodes": req.RoleCodes, "clients": req.Clients})
	response.OK(c, gin.H{"id": fmt.Sprintf("%d", u.ID)})
}

func (h *AdminHandler) syncUserRelations(tx *gorm.DB, userID uint64, roleCodes, clients, whs []string) error {
	if roleCodes != nil {
		_ = tx.Where("user_id = ?", userID).Delete(&iam.UserRole{})
		uid := fmt.Sprintf("%d", userID)
		// clear casbin groupings for user
		roles, _ := h.e.GetRolesForUser("user:" + uid)
		for _, r := range roles {
			_, _ = h.e.DeleteRoleForUser("user:"+uid, r)
		}
		for _, code := range roleCodes {
			var role iam.Role
			if err := tx.Where("code = ?", code).First(&role).Error; err != nil {
				return fmt.Errorf("role not found: %s", code)
			}
			if err := tx.Create(&iam.UserRole{UserID: userID, RoleID: role.ID}).Error; err != nil {
				return err
			}
			if _, err := authz.AddRoleForUser(h.e, uid, code); err != nil {
				return err
			}
		}
		_ = h.e.SavePolicy()
	}
	if clients != nil {
		_ = tx.Where("user_id = ?", userID).Delete(&iam.UserClientAccess{})
		for _, c := range clients {
			if err := tx.Create(&iam.UserClientAccess{UserID: userID, Client: c}).Error; err != nil {
				return err
			}
		}
	}
	if whs != nil {
		_ = tx.Where("user_id = ?", userID).Delete(&iam.UserScope{})
		for _, w := range whs {
			if err := tx.Create(&iam.UserScope{UserID: userID, WarehouseID: w}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *AdminHandler) ListRoles(c *gin.Context) {
	var roles []iam.Role
	if err := h.db.Order("id asc").Find(&roles).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
		return
	}
	type roleVO struct {
		iam.Role
		Permissions []string `json:"permissions"`
	}
	out := make([]roleVO, 0, len(roles))
	for _, r := range roles {
		vo := roleVO{Role: r}
		pols, _ := h.e.GetFilteredPolicy(0, "role:"+r.Code)
		for _, p := range pols {
			if len(p) >= 3 {
				vo.Permissions = append(vo.Permissions, p[1]+":"+p[2])
			}
		}
		out = append(out, vo)
	}
	response.OK(c, gin.H{"list": out})
}

type roleWriteReq struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Remark      string   `json:"remark"`
	Permissions []string `json:"permissions"` // material:view
}

func (h *AdminHandler) CreateRole(c *gin.Context) {
	var req roleWriteReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, "code required")
		return
	}
	role := iam.Role{Code: req.Code, Name: req.Name, Remark: req.Remark}
	if err := h.db.Create(&role).Error; err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	if err := h.replaceRolePerms(req.Code, req.Permissions); err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
		return
	}
	writeAudit(c, h.db, "role.create", "role", fmt.Sprintf("%d", role.ID), gin.H{"code": role.Code, "permissions": req.Permissions})
	response.OK(c, role)
}

func (h *AdminHandler) UpdateRole(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var role iam.Role
	if err := h.db.First(&role, id).Error; err != nil {
		response.Fail(c, http.StatusNotFound, errcode.NotFound, "role not found")
		return
	}
	var req roleWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Remark != "" {
		role.Remark = req.Remark
	}
	if err := h.db.Save(&role).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
		return
	}
	if req.Permissions != nil {
		if err := h.replaceRolePerms(role.Code, req.Permissions); err != nil {
			response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
			return
		}
	}
	writeAudit(c, h.db, "role.update", "role", fmt.Sprintf("%d", role.ID), gin.H{"code": role.Code, "permissions": req.Permissions})
	response.OK(c, role)
}

func (h *AdminHandler) replaceRolePerms(roleCode string, perms []string) error {
	_, _ = h.e.RemoveFilteredPolicy(0, "role:"+roleCode)
	for _, p := range perms {
		parts := strings.SplitN(p, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if _, err := authz.AddRolePermission(h.e, roleCode, parts[0], parts[1]); err != nil {
			return err
		}
	}
	return h.e.SavePolicy()
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	if len(req.NewPassword) < 8 {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, "new password min 8 chars")
		return
	}
	uidStr, _ := c.Get(middleware.CtxUserID)
	uid, _ := strconv.ParseUint(uidStr.(string), 10, 64)
	var user iam.User
	if err := h.db.First(&user, uid).Error; err != nil {
		response.Fail(c, http.StatusNotFound, errcode.NotFound, "user not found")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.AuthDenied, "old password incorrect")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
		return
	}
	user.PasswordHash = string(hash)
	user.MustChangePassword = false
	if err := h.db.Save(&user).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
		return
	}
	_ = h.db.Model(&iam.RefreshToken{}).Where("user_id = ?", user.ID).Update("revoked", true)
	response.OK(c, gin.H{"ok": true})
}
