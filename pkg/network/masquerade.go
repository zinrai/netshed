package network

import (
	"fmt"
	"os/exec"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"github.com/vishvananda/netlink"
)

const (
	commentFmt = "netshed:%s"
	// NFTNL_UDATA_RULE_COMMENT is the TLV type for rule comment
	udataTypeComment = 0x00
)

type MasqueradeManager struct {
	conn *nftables.Conn
}

// encodeComment encodes a comment string into TLV format for nftables UserData
func encodeComment(comment string) []byte {
	// TLV format: Type (1 byte) + Length (1 byte) + Value (comment bytes)
	data := make([]byte, 2+len(comment)+1) // +1 for null terminator
	data[0] = udataTypeComment
	data[1] = byte(len(comment) + 1) // length includes null terminator
	copy(data[2:], comment)
	data[2+len(comment)] = 0x00 // null terminator
	return data
}

// decodeComment extracts comment string from TLV format UserData
func decodeComment(userData []byte) string {
	if len(userData) < 3 {
		return ""
	}
	if userData[0] != udataTypeComment {
		return ""
	}
	length := int(userData[1])
	if len(userData) < 2+length {
		return ""
	}
	// Remove null terminator if present
	comment := userData[2 : 2+length]
	if len(comment) > 0 && comment[len(comment)-1] == 0x00 {
		comment = comment[:len(comment)-1]
	}
	return string(comment)
}

func NewMasqueradeManager() (*MasqueradeManager, error) {
	conn, err := nftables.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize nftables: %v", err)
	}
	return &MasqueradeManager{conn: conn}, nil
}

func enableIPForwarding() error {
	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %v: %s", err, output)
	}
	return nil
}

func (m *MasqueradeManager) ensureNATTable() (*nftables.Table, *nftables.Chain, error) {
	table := m.conn.AddTable(&nftables.Table{
		Family: nftables.TableFamilyIPv4,
		Name:   "nat",
	})

	chain := m.conn.AddChain(&nftables.Chain{
		Name:     "postrouting",
		Table:    table,
		Type:     nftables.ChainTypeNAT,
		Hooknum:  nftables.ChainHookPostrouting,
		Priority: nftables.ChainPriorityNATSource,
	})

	if err := m.conn.Flush(); err != nil {
		return nil, nil, fmt.Errorf("failed to ensure NAT table and chain: %v", err)
	}

	return table, chain, nil
}

func (m *MasqueradeManager) Add(network string) error {
	if err := enableIPForwarding(); err != nil {
		return err
	}

	table, chain, err := m.ensureNATTable()
	if err != nil {
		return err
	}

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

	comment := fmt.Sprintf(commentFmt, network)
	rule := &nftables.Rule{
		Table:    table,
		Chain:    chain,
		UserData: encodeComment(comment),
		Exprs: []expr.Any{
			// Load source address from IPv4 header
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       12,
				Len:          4,
			},
			// Apply netmask
			&expr.Bitwise{
				DestRegister:   1,
				SourceRegister: 1,
				Len:            4,
				Mask:           addrs[0].IPNet.Mask,
				Xor:            []byte{0, 0, 0, 0},
			},
			// Compare with network address
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     addrs[0].IP.Mask(addrs[0].IPNet.Mask),
			},
			// Masquerade action (must be last)
			&expr.Masq{},
		},
	}

	m.conn.AddRule(rule)

	if err := m.conn.Flush(); err != nil {
		return fmt.Errorf("failed to add masquerade rule: %v", err)
	}

	return nil
}

func (m *MasqueradeManager) Remove(network string) error {
	table := &nftables.Table{
		Family: nftables.TableFamilyIPv4,
		Name:   "nat",
	}

	chain := &nftables.Chain{
		Name:  "postrouting",
		Table: table,
	}

	rules, err := m.conn.GetRules(table, chain)
	if err != nil {
		return fmt.Errorf("failed to get rules: %v", err)
	}

	comment := fmt.Sprintf(commentFmt, network)
	for _, rule := range rules {
		if hasComment(rule, comment) && hasMasquerade(rule) {
			if err := m.conn.DelRule(rule); err != nil {
				return fmt.Errorf("failed to delete rule: %v", err)
			}
		}
	}

	if err := m.conn.Flush(); err != nil {
		return fmt.Errorf("failed to remove masquerade rule: %v", err)
	}

	return nil
}

func hasComment(rule *nftables.Rule, comment string) bool {
	return decodeComment(rule.UserData) == comment
}

func hasMasquerade(rule *nftables.Rule) bool {
	for _, e := range rule.Exprs {
		if _, ok := e.(*expr.Masq); ok {
			return true
		}
	}
	return false
}
