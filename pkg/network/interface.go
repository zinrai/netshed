package network

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

type InterfaceManager struct{}

func NewInterfaceManager() *InterfaceManager {
	return &InterfaceManager{}
}

func (m *InterfaceManager) CreateBridge(name string, gateway string) error {
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
		},
	}

	if err := netlink.LinkAdd(bridge); err != nil {
		return fmt.Errorf("failed to create bridge: %v", err)
	}

	addr, err := netlink.ParseAddr(gateway)
	if err != nil {
		return fmt.Errorf("failed to parse gateway address: %v", err)
	}

	if err := netlink.AddrAdd(bridge, addr); err != nil {
		return fmt.Errorf("failed to add address to bridge: %v", err)
	}

	if err := netlink.LinkSetUp(bridge); err != nil {
		return fmt.Errorf("failed to set bridge up: %v", err)
	}

	return nil
}

func (m *InterfaceManager) CreateDummy(name string, address string) error {
	dummy := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
		},
	}

	if err := netlink.LinkAdd(dummy); err != nil {
		return fmt.Errorf("failed to create dummy interface: %v", err)
	}

	addr, err := netlink.ParseAddr(address)
	if err != nil {
		return fmt.Errorf("failed to parse address: %v", err)
	}

	if err := netlink.AddrAdd(dummy, addr); err != nil {
		return fmt.Errorf("failed to add address to dummy interface: %v", err)
	}

	if err := netlink.LinkSetUp(dummy); err != nil {
		return fmt.Errorf("failed to set dummy interface up: %v", err)
	}

	return nil
}

func (m *InterfaceManager) Remove(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("failed to find interface: %v", err)
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to remove interface: %v", err)
	}

	return nil
}
