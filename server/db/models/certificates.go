package models

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Certificate - Certificate database model
type Certificate struct {
	ID             uuid.UUID `gorm:"gorm:"->;<-:create;type:uuid;unique;"`
	CreatedAt      time.Time `gorm:"gorm:"->;<-:create;"`
	CommonName     string    `gorm:"index:unique;"`
	CAType         string
	KeyType        string
	CertificatePEM string
	PrivateKeyPEM  string
}

// BeforeCreate - GORM hook to automatically set values
func (c *Certificate) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	c.CreatedAt = time.Now()
	return nil
}