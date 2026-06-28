package service

import (
	"encoding/json"
	"net"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/tc"
)

func applyClientTCLimit(client model.Client) {
	if err := tc.ApplyClientLimitForCIDRs(client, observedClientCIDRs(client.Email)); err != nil {
		logger.Warningf("[TC] apply client speed limit for %q failed: %v", client.Email, err)
	}
}

func removeClientTCLimit(email string) {
	if err := tc.RemoveClientLimitByEmail(email); err != nil {
		logger.Warningf("[TC] remove client speed limit for %q failed: %v", email, err)
	}
}

func observedClientCIDRs(email string) []string {
	if email == "" {
		return nil
	}
	var row model.InboundClientIps
	if err := database.GetDB().Where("client_email = ?", email).First(&row).Error; err != nil || row.Ips == "" {
		return nil
	}

	cutoff := time.Now().Unix() - clientIpStaleAfterSeconds
	seen := map[string]struct{}{}
	out := []string{}
	add := func(raw string, ts int64) {
		if ts > 0 && ts < cutoff {
			return
		}
		ip := net.ParseIP(raw)
		if ip == nil || ip.To4() == nil {
			return
		}
		cidr := ip.String() + "/32"
		if _, ok := seen[cidr]; ok {
			return
		}
		seen[cidr] = struct{}{}
		out = append(out, cidr)
	}

	var entries []clientIpEntry
	if err := json.Unmarshal([]byte(row.Ips), &entries); err == nil {
		for _, entry := range entries {
			add(entry.IP, entry.Timestamp)
		}
		return out
	}

	var legacy []string
	if err := json.Unmarshal([]byte(row.Ips), &legacy); err == nil {
		for _, ip := range legacy {
			add(ip, 0)
		}
	}
	return out
}
