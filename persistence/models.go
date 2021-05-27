package persistence

import "time"

type Host struct {
	Hostname     string `gorm:"primary_key"`
	LastChecked  time.Time
	Expires      time.Time
	Issuer       string
	Status       string
	ErrorMessage string
}

type NotificationChannel struct {
	Hostname  string `gorm:"primary_key"`
	ChannelID string `gorm:"primary_key"`
	Hosts     []Host `gorm:"foreignKey:Hostname;references:Hostname"`
}
