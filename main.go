package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("subcommand is required: create, delete, or version")
	}

	subcommand := os.Args[1]

	if subcommand == "version" {
		printVersion()
		return
	}

	fs := flag.NewFlagSet(subcommand, flag.ExitOnError)
	configPath := fs.String("config", "", "path to config file")

	if err := fs.Parse(os.Args[2:]); err != nil {
		log.Fatal(err)
	}

	if *configPath == "" {
		log.Fatal("-config flag is required")
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	iface := NewInterfaceManager()
	masq, err := NewMasqueradeManager()
	if err != nil {
		log.Fatal(err)
	}

	switch subcommand {
	case "create":
		runCreate(iface, masq, cfg)
	case "delete":
		runDelete(iface, masq, cfg)
	default:
		log.Fatalf("unknown command: %s. Available commands: create, delete, version", subcommand)
	}
}

func runCreate(iface *InterfaceManager, masq *MasqueradeManager, cfg *Config) {
	for _, n := range cfg.Networks {
		if err := createNetwork(iface, masq, n); err != nil {
			log.Fatalf("failed to create network %s: %v", n.Name, err)
		}
		log.Printf("created network %s", n.Name)
	}
}

func runDelete(iface *InterfaceManager, masq *MasqueradeManager, cfg *Config) {
	for _, n := range cfg.Networks {
		deleteNetwork(iface, masq, n)
		log.Printf("deleted network %s", n.Name)
	}
}

func createNetwork(iface *InterfaceManager, masq *MasqueradeManager, n Network) error {
	switch n.Type {
	case "bridge":
		return createBridge(iface, masq, n)
	case "dummy":
		return iface.CreateDummy(n.Name, n.Address)
	}
	return nil
}

func createBridge(iface *InterfaceManager, masq *MasqueradeManager, n Network) error {
	if err := iface.CreateBridge(n.Name, n.Gateway); err != nil {
		return err
	}
	if !n.Masquerade {
		return nil
	}
	err := masq.Add(n.Name)
	if err == nil {
		return nil
	}
	if rerr := iface.Remove(n.Name); rerr != nil {
		return fmt.Errorf("failed to add masquerade and rollback failed: %v, %v", err, rerr)
	}
	return fmt.Errorf("failed to add masquerade: %v", err)
}

func deleteNetwork(iface *InterfaceManager, masq *MasqueradeManager, n Network) {
	if n.Type == "bridge" && n.Masquerade {
		removeMasquerade(masq, n.Name)
	}
	if err := iface.Remove(n.Name); err != nil {
		log.Fatalf("failed to remove network %s: %v", n.Name, err)
	}
}

func removeMasquerade(masq *MasqueradeManager, name string) {
	if err := masq.Remove(name); err != nil {
		log.Printf("warning: failed to remove masquerade for %s: %v", name, err)
	}
}
