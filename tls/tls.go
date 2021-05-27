package tls

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

func Check(hostname string) (time.Time, string, error) {
	dialer := &net.Dialer{
		Timeout: time.Second,
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", hostname+":443", nil)
	if err != nil {
		return time.Time{}, "", err
	}
	defer conn.Close()

	if err := conn.VerifyHostname(hostname); err != nil {
		return time.Time{}, "", err
	}

	peerCertificates := conn.ConnectionState().PeerCertificates

	notAfter := peerCertificates[0].NotAfter
	issuer := peerCertificates[0].Issuer.Organization[0]

	for _, cert := range peerCertificates {
		logrus.Debugf("Hostname %s, notAfter %s, issuer %s, subject %s", hostname, cert.NotAfter, cert.Issuer, cert.Subject)

		if cert.NotAfter.Before(notAfter) {
			notAfter = cert.NotAfter
		}
	}

	return notAfter, issuer, nil
}
