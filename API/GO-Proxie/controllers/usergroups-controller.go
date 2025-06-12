package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"GO-Proxie/services"
	"gorm.io/gorm"
)

// UserGroup CRUD

func CreateUserGroupHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var group services.UserGroup
		if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
		if err := services.CreateUserGroup(db, &group); err != nil {
			http.Error(w, "Failed to create group", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(group)
	}
}

func GetUserGroupHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}
		group, err := services.GetUserGroup(db, uint(id))
		if err != nil {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(group)
	}
}

func UpdateUserGroupHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var group services.UserGroup
		if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
		if err := services.UpdateUserGroup(db, &group); err != nil {
			http.Error(w, "Failed to update group", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(group)
	}
}

func DeleteUserGroupHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}
		if err := services.DeleteUserGroup(db, uint(id)); err != nil {
			http.Error(w, "Failed to delete group", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
