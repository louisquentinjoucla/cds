package openstack

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// for each ip in the range, look for the first free ones
func (h *HatcheryOpenstack) findAvailableIP(ctx context.Context, workerName string) (string, error) {
	srvs := h.getServers(ctx)

	ipsInfos.mu.Lock()
	defer ipsInfos.mu.Unlock()
	for ip, infos := range ipsInfos.ips {
		// ip used less than 10s
		// 15s max to display worker name on nova list after call create
		if time.Since(infos.dateLastBooked) < 15*time.Second {
			continue
		}
		found := false
		for _, srv := range srvs {
			if infos.workerName == srv.Name {
				found = true
			}
		}
		if !found {
			infos.workerName = ""
			ipsInfos.ips[ip] = infos
		}
	}

	freeIP := []string{}
	for ip := range ipsInfos.ips {
		free := true
		if ipsInfos.ips[ip].workerName != "" {
			continue // ip already used by a worker
		}
		for _, s := range srvs {
			if len(s.Addresses) == 0 {
				continue
			}

			for k, v := range s.Addresses {
				if k != h.Config.NetworkString {
					continue
				}
				switch v.(type) {
				case []interface{}:
					for _, z := range v.([]interface{}) {
						var addr string
						for x, y := range z.(map[string]interface{}) {
							if x == "addr" {
								addr = y.(string)
							}
						}
						if addr == ip {
							free = false
						}
					}
				}
			}

			if !free {
				break
			}
		}
		if free {
			freeIP = append(freeIP, ip)
		}
	}

	if len(freeIP) == 0 {
		return "", fmt.Errorf("No IP left")
	}

	ipToBook := freeIP[rand.Intn(len(freeIP))]
	infos := ipInfos{
		workerName:     workerName,
		dateLastBooked: time.Now(),
	}
	ipsInfos.ips[ipToBook] = infos

	return ipToBook, nil
}
