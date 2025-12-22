package network

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/vishvananda/netlink"
)

const (
	commentFmt = "netshed:%s"
)

type MasqueradeManager struct{}

func NewMasqueradeManager() (*MasqueradeManager, error) {
	return &MasqueradeManager{}, nil
}

func enableIPForwarding() error {
	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %v: %s", err, output)
	}
	return nil
}

func (m *MasqueradeManager) ensureNATTable() error {
	// Add table (idempotent)
	if err := exec.Command("nft", "add", "table", "ip", "nat").Run(); err != nil {
		return fmt.Errorf("failed to add nat table: %v", err)
	}

	// Add chain (idempotent)
	cmd := exec.Command("nft", "add", "chain", "ip", "nat", "postrouting",
		"{ type nat hook postrouting priority srcnat; }")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add postrouting chain: %v", err)
	}

	return nil
}

func (m *MasqueradeManager) Add(network string) error {
	if err := enableIPForwarding(); err != nil {
		return err
	}

	if err := m.ensureNATTable(); err != nil {
		return err
	}

	// Get subnet from interface
	link, err := netlink.LinkByName(network)
	if err != nil {
		return fmt.Errorf("failed to get interface %s: %v", network, err)
	}
	addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to get addresses for %s: %v", network, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("no IPv4 address found for %s", network)
	}

	subnet := addrs[0].IPNet.String()
	comment := fmt.Sprintf(commentFmt, network)

	// Add masquerade rule
	cmd := exec.Command("nft", "add", "rule", "ip", "nat", "postrouting",
		"ip", "saddr", subnet, "masquerade", "comment", fmt.Sprintf(`"%s"`, comment))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add masquerade rule: %v: %s", err, output)
	}

	return nil
}

type nftOutput struct {
	Nftables []struct {
		Rule *struct {
			Handle  int    `json:"handle"`
			Comment string `json:"comment"`
		} `json:"rule,omitempty"`
	} `json:"nftables"`
}

func (m *MasqueradeManager) Remove(network string) error {
	comment := fmt.Sprintf(commentFmt, network)

	// Get rules in JSON format
	cmd := exec.Command("nft", "--json", "list", "chain", "ip", "nat", "postrouting")
	output, err := cmd.Output()
	if err != nil {
		// Chain might not exist, which is fine
		return nil
	}

	var result nftOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("failed to parse nft output: %v", err)
	}

	// Find and delete rules with matching comment
	for _, item := range result.Nftables {
		if item.Rule != nil && item.Rule.Comment == comment {
			delCmd := exec.Command("nft", "delete", "rule", "ip", "nat", "postrouting",
				"handle", fmt.Sprintf("%d", item.Rule.Handle))
			if output, err := delCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to delete rule: %v: %s", err, output)
			}
		}
	}

	return nil
}
