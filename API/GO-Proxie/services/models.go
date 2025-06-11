package services

// User represents the users table
type User struct {
	ID         uint        `gorm:"primaryKey"`
	UUID       string      `gorm:"uniqueIndex"`
	UserGroups []UserGroup `gorm:"many2many:user_user_groups"`
}

// UserGroup represents the user_groups table
type UserGroup struct {
	ID      uint     `gorm:"primaryKey"`
	Name    string   `gorm:"uniqueIndex"`
	Domains []Domain `gorm:"foreignKey:UserGroupID"`
}

// Domain represents the domains table
type Domain struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"uniqueIndex"`
	UserGroupID uint   // Foreign key to UserGroup
}
