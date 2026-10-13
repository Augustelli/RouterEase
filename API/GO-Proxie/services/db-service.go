package services

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitializeDB() *gorm.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL must be configured")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &UserGroup{}, &Domain{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func CreateUser(db *gorm.DB, user *User) error { return db.Create(user).Error }
func GetUser(db *gorm.DB, id uint) (*User, error) {
	var user User
	err := db.First(&user, id).Error
	return &user, err
}
func GetUserByUUID(db *gorm.DB, uuid string) (*User, error) {
	var user User
	err := db.Where("uuid = ?", uuid).First(&user).Error
	return &user, err
}
func UpdateUser(db *gorm.DB, user *User) error { return db.Save(user).Error }
func DeleteUser(db *gorm.DB, id uint) error { return db.Delete(&User{}, id).Error }
func CreateUserGroup(db *gorm.DB, group *UserGroup) error { return db.Create(group).Error }
func GetUserGroup(db *gorm.DB, id uint) (*UserGroup, error) {
	var group UserGroup
	err := db.First(&group, id).Error
	return &group, err
}
func UpdateUserGroup(db *gorm.DB, group *UserGroup) error { return db.Save(group).Error }
func DeleteUserGroup(db *gorm.DB, id uint) error { return db.Delete(&UserGroup{}, id).Error }
func CreateDomain(db *gorm.DB, domain *Domain) error { return db.Create(domain).Error }
func GetDomain(db *gorm.DB, id uint) (*Domain, error) {
	var domain Domain
	err := db.First(&domain, id).Error
	return &domain, err
}
func UpdateDomain(db *gorm.DB, domain *Domain) error { return db.Save(domain).Error }
func DeleteDomain(db *gorm.DB, id uint) error { return db.Delete(&Domain{}, id).Error }
