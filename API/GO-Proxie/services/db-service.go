package services

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

func InitializeDB() *gorm.DB {

	dsn := "host=dns_db user=dns password=password dbname=dns port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// AutoMigrate models
	err = db.AutoMigrate(&User{}, &UserGroup{}, &Domain{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

// CreateUser creates a new user
func CreateUser(db *gorm.DB, user *User) error {
	return db.Create(user).Error
}

// GetUser retrieves a user by ID
func GetUser(db *gorm.DB, id uint) (*User, error) {
	var user User
	err := db.First(&user, id).Error
	return &user, err
}

// GetUserByUUID retrieves a user by UUID
func GetUserByUUID(db *gorm.DB, uuid string) (*User, error) {
	var user User
	err := db.Where("uuid = ?", uuid).First(&user).Error
	return &user, err
}

// UpdateUser updates an existing user
func UpdateUser(db *gorm.DB, user *User) error {
	return db.Save(user).Error
}

// DeleteUser deletes a user by ID
func DeleteUser(db *gorm.DB, id uint) error {
	return db.Delete(&User{}, id).Error
}

// CreateUserGroup creates a new user group
func CreateUserGroup(db *gorm.DB, group *UserGroup) error {
	return db.Create(group).Error
}

// GetUserGroup retrieves a user group by ID
func GetUserGroup(db *gorm.DB, id uint) (*UserGroup, error) {
	var group UserGroup
	err := db.First(&group, id).Error
	return &group, err
}

// UpdateUserGroup updates an existing user group
func UpdateUserGroup(db *gorm.DB, group *UserGroup) error {
	return db.Save(group).Error
}

// DeleteUserGroup deletes a user group by ID
func DeleteUserGroup(db *gorm.DB, id uint) error {
	return db.Delete(&UserGroup{}, id).Error
}

// CreateDomain creates a new domain
func CreateDomain(db *gorm.DB, domain *Domain) error {
	return db.Create(domain).Error
}

// GetDomain retrieves a domain by ID
func GetDomain(db *gorm.DB, id uint) (*Domain, error) {
	var domain Domain
	err := db.First(&domain, id).Error
	return &domain, err
}

// UpdateDomain updates an existing domain
func UpdateDomain(db *gorm.DB, domain *Domain) error {
	return db.Save(domain).Error
}

// DeleteDomain deletes a domain by ID
func DeleteDomain(db *gorm.DB, id uint) error {
	return db.Delete(&Domain{}, id).Error
}
