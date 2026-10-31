package models

import "time"

// No names, emails, full URLs, IPs or raw user IDs are retained here.
type CampaignEvent struct {
	ID           string    `gorm:"primaryKey;size:80" json:"-"`
	VisitID      string    `gorm:"size:36;index" json:"-"`
	Kind         string    `gorm:"size:20" json:"-"`
	Source       string    `gorm:"size:100" json:"-"`
	Medium       string    `gorm:"size:100" json:"-"`
	CampaignName string    `gorm:"size:100" json:"-"`
	Content      string    `gorm:"size:100" json:"-"`
	Term         string    `gorm:"size:100" json:"-"`
	CreatedAt    time.Time `gorm:"index" json:"-"`
}
