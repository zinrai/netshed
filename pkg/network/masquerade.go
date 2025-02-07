package network

import (
	"fmt"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
)

const (
	commentFmt = "netshed:%s"
)

type MasqueradeManager struct {
	conn *nftables.Conn
}

func NewMasqueradeManager() (*MasqueradeManager, error) {
	conn, err := nftables.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize nftables: %v", err)
	}
	return &MasqueradeManager{conn: conn}, nil
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
	table, chain, err := m.ensureNATTable()
	if err != nil {
		return err
	}

	rule := &nftables.Rule{
		Table: table,
		Chain: chain,
		Exprs: []expr.Any{
			&expr.Meta{Key: expr.MetaKeyOIFNAME, Register: 1},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte(network),
			},
			&expr.Masq{},
			&expr.Meta{Key: expr.MetaKeyIIFNAME, Register: 1},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte(fmt.Sprintf(commentFmt, network)),
			},
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
	for _, e := range rule.Exprs {
		if cmp, ok := e.(*expr.Cmp); ok {
			if string(cmp.Data) == comment {
				return true
			}
		}
	}
	return false
}

func hasMasquerade(rule *nftables.Rule) bool {
	for _, e := range rule.Exprs {
		if _, ok := e.(*expr.Masq); ok {
			return true
		}
	}
	return false
}
