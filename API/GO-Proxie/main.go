package main

import (
	"GO-Proxie/controllers"
	"GO-Proxie/services"
	"net/http"
)

func main() {
	db := services.InitializeDB()

	http.HandleFunc("/users/create", controllers.CreateUserHandler(db))
	http.HandleFunc("/users/get", controllers.GetUserHandler(db))
	http.HandleFunc("/users/update", controllers.UpdateUserHandler(db))
	http.HandleFunc("/users/delete", controllers.DeleteUserHandler(db))

	http.HandleFunc("/groups/create", controllers.CreateUserGroupHandler(db))
	http.HandleFunc("/groups/get", controllers.GetUserGroupHandler(db))
	http.HandleFunc("/groups/update", controllers.UpdateUserGroupHandler(db))
	http.HandleFunc("/groups/delete", controllers.DeleteUserGroupHandler(db))

	http.HandleFunc("/domains/create", controllers.CreateDomainHandler(db))
	http.HandleFunc("/domains/get", controllers.GetDomainHandler(db))
	http.HandleFunc("/domains/update", controllers.UpdateDomainHandler(db))
	http.HandleFunc("/domains/delete", controllers.DeleteDomainHandler(db))

	http.ListenAndServe(":8087", nil)
}
us4r