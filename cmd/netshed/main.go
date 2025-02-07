package main

import (
	"flag"
	"log"
	"os"

	"github.com/zinrai/netshed/pkg/config"
	"github.com/zinrai/netshed/pkg/network"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("subcommand is required: create or delete")
	}

	subcommand := os.Args[1]

	fs := flag.NewFlagSet(subcommand, flag.ExitOnError)
	configPath := fs.String("config", "", "path to config file")

	if err := fs.Parse(os.Args[2:]); err != nil {
		log.Fatal(err)
	}

	if *configPath == "" {
		log.Fatal("-config flag is required")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	iface := network.NewInterfaceManager()
	masq, err := network.NewMasqueradeManager()
	if err != nil {
		log.Fatal(err)
	}

	switch subcommand {
	case "create":
		for _, net := range cfg.Networks {
			switch net.Type {
			case "bridge":
				if err := iface.CreateBridge(net.Name, net.Gateway); err != nil {
					log.Fatalf("failed to create bridge %s: %v", net.Name, err)
				}
				if net.Masquerade {
					if err := masq.Add(net.Name); err != nil {
						// ロールバック
						if rerr := iface.Remove(net.Name); rerr != nil {
							log.Fatalf("failed to add masquerade and rollback failed: %v, %v", err, rerr)
						}
						log.Fatalf("failed to add masquerade: %v", err)
					}
				}
			case "dummy":
				if err := iface.CreateDummy(net.Name, net.Address); err != nil {
					log.Fatalf("failed to create dummy interface %s: %v", net.Name, err)
				}
			}
			log.Printf("created network %s", net.Name)
		}

	case "delete":
		for _, net := range cfg.Networks {
			if net.Type == "bridge" && net.Masquerade {
				if err := masq.Remove(net.Name); err != nil {
					log.Printf("warning: failed to remove masquerade for %s: %v", net.Name, err)
				}
			}
			if err := iface.Remove(net.Name); err != nil {
				log.Fatalf("failed to remove network %s: %v", net.Name, err)
			}
			log.Printf("deleted network %s", net.Name)
		}

	default:
		log.Fatalf("unknown command: %s. Available commands: create, delete", subcommand)
	}
}
