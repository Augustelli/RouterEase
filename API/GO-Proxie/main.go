package main

import (
	"GO-Proxie/controllers"
	"GO-Proxie/services"
	"log"
	"net/http"
)

func main() {

	db := services.InitializeDB()
	// User endpoints
	http.HandleFunc("/users/create", controllers.CreateUserHandler(db))
	http.HandleFunc("/users/get", controllers.GetUserHandler(db))
	http.HandleFunc("/users/update", controllers.UpdateUserHandler(db))
	http.HandleFunc("/users/delete", controllers.DeleteUserHandler(db))

	// UserGroups endpoints
	http.HandleFunc("/groups/create", controllers.CreateUserGroupHandler(db))
	http.HandleFunc("/groups/get", controllers.GetUserGroupHandler(db))
	http.HandleFunc("/groups/update", controllers.UpdateUserGroupHandler(db))
	http.HandleFunc("/groups/delete", controllers.DeleteUserGroupHandler(db))

	// Domain endpoints
	http.HandleFunc("/domains/create", controllers.CreateDomainHandler(db))
	http.HandleFunc("/domains/get", controllers.GetDomainHandler(db))
	http.HandleFunc("/domains/update", controllers.UpdateDomainHandler(db))
	http.HandleFunc("/domains/delete", controllers.DeleteDomainHandler(db))

	http.HandleFunc("/dns-query", services.DoHandler)

	log.Println("Starting proxy on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
