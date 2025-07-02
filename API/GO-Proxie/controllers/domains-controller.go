package controllers

import (
	"GO-Proxie/services"
	"encoding/json"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

func CreateDomainHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var domain services.Domain
		if err := json.NewDecoder(r.Body).Decode(&domain); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
		if err := services.CreateDomain(db, &domain); err != nil {
			http.Error(w, "Failed to create domain", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(domain)
	}
}

func GetDomainHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}
		domain, err := services.GetDomain(db, uint(id))
		if err != nil {
			http.Error(w, "Domain not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(domain)
	}
}

func UpdateDomainHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var domain services.Domain
		if err := json.NewDecoder(r.Body).Decode(&domain); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
		if err := services.UpdateDomain(db, &domain); err != nil {
			http.Error(w, "Failed to update domain", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(domain)
	}
}

func DeleteDomainHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}
		if err := services.DeleteDomain(db, uint(id)); err != nil {
			http.Error(w, "Failed to delete domain", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
