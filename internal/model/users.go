package model

import (
	"github.com/go-dev-frame/sponge/pkg/sgorm"
	"time"
)

type Users struct {
	sgorm.Model `gorm:"embedded"` // embed id and time

	Email               string    `gorm:"column:email;type:varchar(100);not null" json:"email"`
	EncryptedPassword   string    `gorm:"column:encrypted_password;type:varchar(100);not null" json:"encryptedPassword"`
	ResetPasswordToken  string    `gorm:"column:reset_password_token;type:varchar(100)" json:"resetPasswordToken"`
	ResetPasswordSentAt time.Time `gorm:"column:reset_password_sent_at;type:datetime" json:"resetPasswordSentAt"`
	RememberCreatedAt   time.Time `gorm:"column:remember_created_at;type:datetime" json:"rememberCreatedAt"`
	SignInCount         int       `gorm:"column:sign_in_count;type:int(11);not null" json:"signInCount"`
	CurrentSignInAt     time.Time `gorm:"column:current_sign_in_at;type:datetime" json:"currentSignInAt"`
	LastSignInAt        time.Time `gorm:"column:last_sign_in_at;type:datetime" json:"lastSignInAt"`
	CurrentSignInIP     string    `gorm:"column:current_sign_in_ip;type:varchar(100)" json:"currentSignInIP"`
	LastSignInIP        string    `gorm:"column:last_sign_in_ip;type:varchar(100)" json:"lastSignInIP"`
	ConfirmationToken   string    `gorm:"column:confirmation_token;type:varchar(100)" json:"confirmationToken"`
	ConfirmedAt         time.Time `gorm:"column:confirmed_at;type:datetime" json:"confirmedAt"`
	ConfirmationSentAt  time.Time `gorm:"column:confirmation_sent_at;type:datetime" json:"confirmationSentAt"`
	UnconfirmedEmail    string    `gorm:"column:unconfirmed_email;type:varchar(100)" json:"unconfirmedEmail"`
	FailedAttempts      int       `gorm:"column:failed_attempts;type:int(11);not null" json:"failedAttempts"`
	UnlockToken         string    `gorm:"column:unlock_token;type:varchar(100)" json:"unlockToken"`
	LockedAt            time.Time `gorm:"column:locked_at;type:datetime" json:"lockedAt"`
	Admin               int       `gorm:"column:admin;type:tinyint(4)" json:"admin"`
	Username            string    `gorm:"column:username;type:varchar(100)" json:"username"`
	RememberToken       string    `gorm:"column:remember_token;type:varchar(100)" json:"rememberToken"`
}

// UsersColumnNames Whitelist for custom query fields to prevent sql injection attacks
var UsersColumnNames = map[string]bool{
	"id":                     true,
	"created_at":             true,
	"updated_at":             true,
	"deleted_at":             true,
	"email":                  true,
	"encrypted_password":     true,
	"reset_password_token":   true,
	"reset_password_sent_at": true,
	"remember_created_at":    true,
	"sign_in_count":          true,
	"current_sign_in_at":     true,
	"last_sign_in_at":        true,
	"current_sign_in_ip":     true,
	"last_sign_in_ip":        true,
	"confirmation_token":     true,
	"confirmed_at":           true,
	"confirmation_sent_at":   true,
	"unconfirmed_email":      true,
	"failed_attempts":        true,
	"unlock_token":           true,
	"locked_at":              true,
	"admin":                  true,
	"username":               true,
	"remember_token":         true,
}
