package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lagren/tlsguard/persistence"
	"github.com/lagren/tlsguard/slack"
	"github.com/lagren/tlsguard/tls"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type tlsChecker struct {
	db *gorm.DB
}

func (c *tlsChecker) Add(ctx context.Context, hostname string) error {
	h := &persistence.Host{
		Hostname: hostname,
	}

	if err := c.db.WithContext(ctx).FirstOrCreate(&h, "hostname = ?", hostname).Error; err != nil {
		return err
	}

	if err := c.Update(ctx, hostname); err != nil {
		logrus.Warnf("Could not update %s: %s", hostname, err)
	}

	return nil
}

func (c *tlsChecker) Subscribe(ctx context.Context, hostname string, channelID string) error {
	nc := &persistence.NotificationChannel{
		Hostname:  hostname,
		ChannelID: channelID,
	}

	err := c.db.WithContext(ctx).
		FirstOrCreate(nc, "hostname = ? AND channel_id = ?", hostname, channelID).Error

	return err
}

func (c *tlsChecker) AddAndSubscribe(ctx context.Context, channelID string, hostname string) error {
	err := c.Add(ctx, hostname)

	if err != nil {
		return err
	}

	err = c.Subscribe(ctx, hostname, channelID)

	if err != nil {
		return err
	}

	return nil
}

func (c *tlsChecker) Remove(ctx context.Context, channelID, hostname string) error {
	err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := c.db.WithContext(ctx).Delete(&persistence.NotificationChannel{}, "hostname = ? AND channel_id = ?", hostname, channelID).Error; err != nil {
			logrus.Errorf("Could not delete notification channel entry: %s", err)

			return fmt.Errorf("could not delete notification channel entry")
		}

		// TODO Only remove host if no one is subscribing to it

		if err := c.db.WithContext(ctx).Delete(&persistence.Host{}, "hostname = ?", hostname).Error; err != nil {
			logrus.Errorf("Could not delete host entry: %s", err)

			return fmt.Errorf("could not delete host entry")
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("could not remove: %w", err)
	}

	return nil
}

func (c *tlsChecker) Update(ctx context.Context, hostname string) error {
	var host persistence.Host

	err := c.db.WithContext(ctx).Find(&host, "hostname = ?", hostname).Error
	if err != nil {
		return err
	}

	expires, issuer, err := tls.Check(hostname)

	if err != nil {
		return err
	}

	host.Expires = expires
	host.Issuer = issuer

	return c.db.Save(host).Error
}

func (c *tlsChecker) Run(ctx context.Context) error {
	logrus.Infof("Initiate scan run...")
	defer logrus.Infof("Scan run finished")

	var hosts []persistence.Host

	err := c.db.WithContext(ctx).Find(&hosts).Error
	if err != nil {
		return err
	}

	for _, host := range hosts {
		logrus.Infof("Checking %s...", host.Hostname)

		expires, issuer, err := tls.Check(host.Hostname)

		if err != nil {
			host.Status = "error"
			host.ErrorMessage = err.Error()
		} else {
			host.Status = "valid"
			host.ErrorMessage = ""
		}

		host.Expires = expires
		host.Issuer = issuer
		host.LastChecked = time.Now()

		if err := c.db.Save(host).Error; err != nil {
			return err
		}

		if err != nil {
			return err
		}
	}

	var channelIDs []string

	err = c.db.Model(&persistence.NotificationChannel{}).Distinct().Pluck("channel_id", &channelIDs).Error
	if err != nil {
		return err
	}

	var ncs []persistence.NotificationChannel
	if err := c.db.Preload("Hosts").Find(&ncs).Error; err != nil {
		return err
	}

	for _, nc := range ncs {
		for _, h := range nc.Hosts {
			logrus.Infof("Host %s - %s", h.Hostname, h.Expires)
			if err := slack.PostMessage(ctx, nc.ChannelID, h.Hostname, h.Expires); err != nil {
				return err
			}
		}

	}

	return nil
}

func (c *tlsChecker) DailyRun(ctx context.Context) error {
	logrus.Infof("Initiate scan run...")
	defer logrus.Infof("Scan run finished")

	var hosts []persistence.Host

	err := c.db.WithContext(ctx).Find(&hosts).Error
	if err != nil {
		return err
	}

	for _, host := range hosts {
		logrus.Infof("Checking %s...", host.Hostname)

		expires, issuer, err := tls.Check(host.Hostname)

		if err != nil {
			host.Status = "error"
			host.ErrorMessage = err.Error()
		} else {
			host.Status = "valid"
			host.ErrorMessage = ""
		}

		host.Expires = expires
		host.Issuer = issuer
		host.LastChecked = time.Now()

		if err := c.db.Save(host).Error; err != nil {
			return err
		}

		if err != nil {
			return err
		}

		if expires.Before(time.Now().AddDate(0, 0, 21)) {
			var channels []persistence.NotificationChannel

			err := c.db.Find(&channels, "hostname = ?", host.Hostname).Error

			if err != nil {
				return err
			}

			for _, channel := range channels {
				if err := slack.PostMessage(ctx, channel.ChannelID, host.Hostname, expires); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
