package controllers

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"GO-Proxie/services"
	"gorm.io/gorm"
)

const maxJSONBytes = 1 << 20

type V1API struct {
	DB     *gorm.DB
	Engine *services.PolicyEngine
}

func (a V1API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/"), "/")
	parts := strings.Split(path, "/")
	if path == "dns/status" && r.Method == http.MethodGet {
		writeJSON(w, 200, map[string]string{"status": "ready"})
		return
	}
	if len(parts) == 1 && parts[0] == "routers" {
		a.routers(w, r)
		return
	}
	if len(parts) >= 2 && parts[0] == "routers" {
		a.router(w, r, parts[1], parts[2:])
		return
	}
	if len(parts) == 1 && parts[0] == "policies" {
		a.policies(w, r)
		return
	}
	if len(parts) >= 2 && parts[0] == "policies" {
		a.policy(w, r, parts[1], parts[2:])
		return
	}
	if len(parts) >= 2 && parts[0] == "groups" {
		a.group(w, r, parts[1], parts[2:])
		return
	}
	if len(parts) >= 2 && parts[0] == "devices" {
		a.device(w, r, parts[1], parts[2:])
		return
	}
	http.NotFound(w, r)
}

func (a V1API) routers(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		var rows []services.Router
		if err := a.DB.Order("created_at").Find(&rows).Error; err != nil {
			dbError(w, err)
			return
		}
		writeJSON(w, 200, rows)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &input) || !validName(input.Name) {
		badRequest(w, "valid name required")
		return
	}
	row := services.Router{Name: strings.TrimSpace(input.Name), Status: "active"}
	if err := a.DB.Create(&row).Error; err != nil {
		dbError(w, err)
		return
	}
	writeJSON(w, 201, row)
}
func (a V1API) router(w http.ResponseWriter, r *http.Request, id string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			var row services.Router
			if err := a.DB.First(&row, "id = ?", id).Error; err != nil {
				dbError(w, err)
				return
			}
			writeJSON(w, 200, row)
			return
		}
		if r.Method == http.MethodDelete {
			if err := a.DB.Delete(&services.Router{}, "id = ?", id).Error; err != nil {
				dbError(w, err)
				return
			}
			a.Engine.Invalidate()
			w.WriteHeader(204)
			return
		}
		if r.Method == http.MethodPut {
			var input struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			}
			if !decode(w, r, &input) || !validName(input.Name) || (input.Status != "active" && input.Status != "disabled") {
				badRequest(w, "valid name and status required")
				return
			}
			if err := a.DB.Model(&services.Router{}).Where("id = ?", id).Updates(map[string]interface{}{"name": input.Name, "status": input.Status, "updated_at": time.Now().UTC()}).Error; err != nil {
				dbError(w, err)
				return
			}
			w.WriteHeader(204)
			return
		}
		methodNotAllowed(w)
		return
	}
	if tail[0] == "keys" {
		a.routerKeys(w, r, id, tail[1:])
		return
	}
	if tail[0] == "devices" {
		a.routerDevices(w, r, id, tail[1:])
		return
	}
	if tail[0] == "groups" {
		a.routerGroups(w, r, id, tail[1:])
		return
	}
	http.NotFound(w, r)
}
func (a V1API) routerKeys(w http.ResponseWriter, r *http.Request, routerID string, tail []string) {
	if len(tail) == 0 && r.Method == http.MethodPost {
		var in struct {
			Scopes    []string   `json:"scopes"`
			ExpiresAt *time.Time `json:"expires_at"`
		}
		if !decode(w, r, &in) {
			return
		}
		key, raw, err := services.IssueRouterKey(a.DB, routerID, in.Scopes, in.ExpiresAt)
		if err != nil {
			dbError(w, err)
			return
		}
		writeJSON(w, 201, map[string]interface{}{"key": key, "api_key": raw})
		return
	}
	if len(tail) == 1 && r.Method == http.MethodDelete {
		if err := services.RevokeRouterKey(a.DB, tail[0]); err != nil {
			dbError(w, err)
			return
		}
		w.WriteHeader(204)
		return
	}
	if len(tail) == 1 {
		a.device(w, r, tail[0], nil)
		return
	}
	methodNotAllowed(w)
}
func (a V1API) routerDevices(w http.ResponseWriter, r *http.Request, routerID string, tail []string) {
	if len(tail) == 0 && r.Method == http.MethodGet {
		var rows []services.Device
		if err := a.DB.Where("router_id = ?", routerID).Order("name").Find(&rows).Error; err != nil {
			dbError(w, err)
			return
		}
		writeJSON(w, 200, rows)
		return
	}
	if len(tail) == 0 && r.Method == http.MethodPost {
		var in struct {
			Name       string `json:"name"`
			MACAddress string `json:"mac_address"`
		}
		if !decode(w, r, &in) || !validName(in.Name) || !validMAC(in.MACAddress) {
			badRequest(w, "name and canonical MAC address required")
			return
		}
		row := services.Device{RouterID: routerID, Name: strings.TrimSpace(in.Name), MACAddress: strings.ToLower(in.MACAddress)}
		if err := a.DB.Create(&row).Error; err != nil {
			dbError(w, err)
			return
		}
		a.Engine.Invalidate()
		writeJSON(w, 201, row)
		return
	}
	if len(tail) == 1 {
		a.group(w, r, tail[0], nil)
		return
	}
	methodNotAllowed(w)
}
func (a V1API) routerGroups(w http.ResponseWriter, r *http.Request, routerID string, tail []string) {
	if len(tail) == 0 && r.Method == http.MethodGet {
		var rows []services.DeviceGroup
		if err := a.DB.Where("router_id = ?", routerID).Order("name").Find(&rows).Error; err != nil {
			dbError(w, err)
			return
		}
		writeJSON(w, 200, rows)
		return
	}
	if len(tail) == 0 && r.Method == http.MethodPost {
		var in struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if !decode(w, r, &in) || !validName(in.Name) {
			badRequest(w, "valid name required")
			return
		}
		row := services.DeviceGroup{RouterID: routerID, Name: strings.TrimSpace(in.Name), Description: in.Description}
		if err := a.DB.Create(&row).Error; err != nil {
			dbError(w, err)
			return
		}
		a.Engine.Invalidate()
		writeJSON(w, 201, row)
		return
	}
	methodNotAllowed(w)
}
func (a V1API) policies(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		var rows []services.DNSPolicy
		if err := a.DB.Order("name").Find(&rows).Error; err != nil {
			dbError(w, err)
			return
		}
		writeJSON(w, 200, rows)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var row services.DNSPolicy
	if !decode(w, r, &row) || !validName(row.Name) || !validAction(row.DefaultAction) {
		badRequest(w, "name and default_action ALLOW/BLOCK required")
		return
	}
	if err := a.DB.Create(&row).Error; err != nil {
		dbError(w, err)
		return
	}
	a.Engine.Invalidate()
	writeJSON(w, 201, row)
}
func (a V1API) policy(w http.ResponseWriter, r *http.Request, id string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			var row services.DNSPolicy
			if err := a.DB.First(&row, "id = ?", id).Error; err != nil {
				dbError(w, err)
				return
			}
			writeJSON(w, 200, row)
			return
		}
		if r.Method == http.MethodPut {
			var row services.DNSPolicy
			if !decode(w, r, &row) || !validName(row.Name) || !validAction(row.DefaultAction) {
				badRequest(w, "valid policy required")
				return
			}
			if err := a.DB.Model(&services.DNSPolicy{}).Where("id = ?", id).Updates(map[string]interface{}{"name": row.Name, "description": row.Description, "enabled": row.Enabled, "default_action": row.DefaultAction, "updated_at": time.Now().UTC()}).Error; err != nil {
				dbError(w, err)
				return
			}
			a.Engine.Invalidate()
			w.WriteHeader(204)
			return
		}
		if r.Method == http.MethodDelete {
			if err := a.DB.Delete(&services.DNSPolicy{}, "id = ?", id).Error; err != nil {
				dbError(w, err)
				return
			}
			a.Engine.Invalidate()
			w.WriteHeader(204)
			return
		}
		methodNotAllowed(w)
		return
	}
	if tail[0] == "rules" {
		a.rules(w, r, id, tail[1:])
		return
	}
	if tail[0] == "assignments" {
		a.assignments(w, r, id, tail[1:])
		return
	}
	http.NotFound(w, r)
}
func (a V1API) rules(w http.ResponseWriter, r *http.Request, policyID string, tail []string) {
	if len(tail) == 0 && r.Method == http.MethodGet {
		var rows []services.DNSRule
		if err := a.DB.Where("policy_id = ?", policyID).Find(&rows).Error; err != nil {
			dbError(w, err)
			return
		}
		writeJSON(w, 200, rows)
		return
	}
	if len(tail) == 0 && r.Method == http.MethodPost {
		var row services.DNSRule
		if !decode(w, r, &row) {
			return
		}
		domain, err := normalizeForAPI(row.Domain)
		if err != nil || !validAction(row.Action) {
			badRequest(w, "valid domain and action required")
			return
		}
		row.PolicyID = policyID
		row.Domain = domain
		if err := a.DB.Create(&row).Error; err != nil {
			dbError(w, err)
			return
		}
		a.Engine.Invalidate()
		writeJSON(w, 201, row)
		return
	}
	if len(tail) == 1 && r.Method == http.MethodDelete {
		if err := a.DB.Delete(&services.DNSRule{}, "id = ? AND policy_id = ?", tail[0], policyID).Error; err != nil {
			dbError(w, err)
			return
		}
		a.Engine.Invalidate()
		w.WriteHeader(204)
		return
	}
	if len(tail) == 1 && r.Method == http.MethodPut {
		var row services.DNSRule
		if !decode(w, r, &row) {
			return
		}
		domain, err := normalizeForAPI(row.Domain)
		if err != nil || !validAction(row.Action) {
			badRequest(w, "valid domain and action required")
			return
		}
		if err := a.DB.Model(&services.DNSRule{}).Where("id = ? AND policy_id = ?", tail[0], policyID).Updates(map[string]interface{}{"domain": domain, "action": row.Action, "enabled": row.Enabled, "updated_at": time.Now().UTC()}).Error; err != nil {
			dbError(w, err)
			return
		}
		a.Engine.Invalidate()
		w.WriteHeader(204)
		return
	}
	methodNotAllowed(w)
}
func (a V1API) assignments(w http.ResponseWriter, r *http.Request, policyID string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodDelete {
		if err := a.DB.Delete(&services.PolicyAssignment{}, "id = ? AND policy_id = ?", tail[0], policyID).Error; err != nil {
			dbError(w, err)
			return
		}
		a.Engine.Invalidate()
		w.WriteHeader(204)
		return
	}
	if len(tail) != 0 {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodGet {
		var rows []services.PolicyAssignment
		if err := a.DB.Where("policy_id = ?", policyID).Find(&rows).Error; err != nil {
			dbError(w, err)
			return
		}
		writeJSON(w, 200, rows)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var row services.PolicyAssignment
	if !decode(w, r, &row) || (row.DeviceID == nil && row.GroupID == nil) || (row.DeviceID != nil && row.GroupID != nil) || row.Priority < 0 {
		badRequest(w, "exactly one target and non-negative priority required")
		return
	}
	row.PolicyID = policyID
	if err := a.DB.Create(&row).Error; err != nil {
		dbError(w, err)
		return
	}
	a.Engine.Invalidate()
	writeJSON(w, 201, row)
}
func (a V1API) group(w http.ResponseWriter, r *http.Request, id string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			var row services.DeviceGroup
			if err := a.DB.First(&row, "id = ?", id).Error; err != nil {
				dbError(w, err)
				return
			}
			writeJSON(w, 200, row)
			return
		}
		if r.Method == http.MethodPut {
			var row services.DeviceGroup
			if !decode(w, r, &row) || !validName(row.Name) {
				badRequest(w, "valid group required")
				return
			}
			if err := a.DB.Model(&services.DeviceGroup{}).Where("id=?", id).Updates(map[string]interface{}{"name": row.Name, "description": row.Description, "updated_at": time.Now().UTC()}).Error; err != nil {
				dbError(w, err)
				return
			}
			a.Engine.Invalidate()
			w.WriteHeader(204)
			return
		}
		if r.Method == http.MethodDelete {
			if err := a.DB.Delete(&services.DeviceGroup{}, "id=?", id).Error; err != nil {
				dbError(w, err)
				return
			}
			a.Engine.Invalidate()
			w.WriteHeader(204)
			return
		}
	}
	if len(tail) == 2 && tail[0] == "devices" {
		if r.Method == http.MethodPost {
			if err := a.DB.Exec("INSERT INTO device_group_members(device_id,group_id) VALUES (?,?) ON CONFLICT DO NOTHING", tail[1], id).Error; err != nil {
				dbError(w, err)
				return
			}
			a.Engine.Invalidate()
			w.WriteHeader(204)
			return
		}
		if r.Method == http.MethodDelete {
			if err := a.DB.Exec("DELETE FROM device_group_members WHERE device_id=? AND group_id=?", tail[1], id).Error; err != nil {
				dbError(w, err)
				return
			}
			a.Engine.Invalidate()
			w.WriteHeader(204)
			return
		}
	}
	methodNotAllowed(w)
}
func (a V1API) device(w http.ResponseWriter, r *http.Request, id string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			var row services.Device
			if err := a.DB.First(&row, "id=?", id).Error; err != nil {
				dbError(w, err)
				return
			}
			writeJSON(w, 200, row)
			return
		}
		if r.Method == http.MethodPut {
			var in struct {
				Name       string `json:"name"`
				MACAddress string `json:"mac_address"`
			}
			if !decode(w, r, &in) || !validName(in.Name) || !validMAC(in.MACAddress) {
				badRequest(w, "valid device required")
				return
			}
			if err := a.DB.Model(&services.Device{}).Where("id=?", id).Updates(map[string]interface{}{"name": in.Name, "mac_address": strings.ToLower(in.MACAddress), "updated_at": time.Now().UTC()}).Error; err != nil {
				dbError(w, err)
				return
			}
			a.Engine.Invalidate()
			w.WriteHeader(204)
			return
		}
		if r.Method == http.MethodDelete {
			if err := a.DB.Delete(&services.Device{}, "id=?", id).Error; err != nil {
				dbError(w, err)
				return
			}
			a.Engine.Invalidate()
			w.WriteHeader(204)
			return
		}
	}
	if len(tail) == 1 && tail[0] == "effective-policy" && r.Method == http.MethodGet {
		d, err := a.Engine.Evaluate(r.Context(), id, r.URL.Query().Get("domain"))
		if err != nil {
			http.Error(w, "policy store unavailable", 503)
			return
		}
		writeJSON(w, 200, d)
		return
	}
	methodNotAllowed(w)
}

func decode(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes)
	defer r.Body.Close()
	if json.NewDecoder(r.Body).Decode(target) != nil {
		badRequest(w, "invalid JSON")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
func badRequest(w http.ResponseWriter, message string) { http.Error(w, message, http.StatusBadRequest) }
func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
func dbError(w http.ResponseWriter, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "not found", 404)
		return
	}
	http.Error(w, "database operation failed", 500)
}
func validName(v string) bool   { return len(strings.TrimSpace(v)) > 0 && len(v) <= 128 }
func validAction(v string) bool { return v == "ALLOW" || v == "BLOCK" }
func validMAC(v string) bool    { _, err := net.ParseMAC(v); return err == nil && len(v) == 17 }
func normalizeForAPI(v string) (string, error) {
	v = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(v), "."))
	if v == "" || len(v) > 253 {
		return "", errors.New("invalid domain")
	}
	for _, p := range strings.Split(v, ".") {
		if p == "" || len(p) > 63 {
			return "", errors.New("invalid domain")
		}
	}
	return v, nil
}

type RouterAPI struct {
	DB     *gorm.DB
	Engine *services.PolicyEngine
}

func (a RouterAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/router/leases" || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	raw := r.Header.Get("X-Router-API-Key")
	key, err := services.AuthenticateRouterKey(a.DB, raw, "inventory:write")
	if err != nil {
		http.Error(w, "router authorization required", 401)
		return
	}
	var input struct {
		Leases []struct {
			DeviceID         string `json:"device_id"`
			IP               string `json:"ip"`
			ExpiresInSeconds int    `json:"expires_in_seconds"`
		} `json:"leases"`
	}
	if !decode(w, r, &input) || len(input.Leases) == 0 || len(input.Leases) > 512 {
		badRequest(w, "1..512 leases required")
		return
	}
	now := time.Now().UTC()
	for _, lease := range input.Leases {
		if net.ParseIP(lease.IP) == nil || lease.ExpiresInSeconds < 1 || lease.ExpiresInSeconds > 900 {
			badRequest(w, "invalid lease")
			return
		}
		var device services.Device
		if err := a.DB.Where("id = ? AND router_id = ?", lease.DeviceID, key.RouterID).First(&device).Error; err != nil {
			http.Error(w, "unknown router-scoped device", 403)
			return
		}
		expires := now.Add(time.Duration(lease.ExpiresInSeconds) * time.Second)
		if err := a.DB.Exec(`INSERT INTO device_leases(router_id,device_id,ip,observed_at,expires_at) VALUES(?,?,?,?,?) ON CONFLICT (router_id,ip) DO UPDATE SET device_id=excluded.device_id,observed_at=excluded.observed_at,expires_at=excluded.expires_at`, key.RouterID, lease.DeviceID, lease.IP, now, expires).Error; err != nil {
			dbError(w, err)
			return
		}
		_ = a.DB.Model(&services.Device{}).Where("id=?", lease.DeviceID).Updates(map[string]interface{}{"last_known_ip": lease.IP, "last_seen": now}).Error
	}
	a.Engine.Invalidate()
	w.WriteHeader(204)
}
