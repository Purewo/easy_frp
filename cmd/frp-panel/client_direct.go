package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	localclient "frp-ui-backend/internal/client"
)

type directClientOptions struct {
	data       string
	frpcPath   string
	workDir    string
	server     string
	deviceName string
	jsonOut    bool
	detach     bool
}

func isClientDirectCommand(cmd string) bool {
	switch cmd {
	case "host", "join", "rooms", "list":
		return true
	default:
		return false
	}
}

func runClientDirect(args []string) error {
	switch args[0] {
	case "host":
		return runClientHost(args[1:])
	case "join":
		return runClientJoin(args[1:])
	case "rooms", "list":
		return runClientRooms(args[1:])
	default:
		return fmt.Errorf("unknown client command %q", args[0])
	}
}

func addDirectClientFlags(fs *flag.FlagSet, opts *directClientOptions) {
	fs.StringVar(&opts.data, "data", filepath.Join("data", "client.json"), "client JSON data file")
	fs.StringVar(&opts.frpcPath, "frpc", "frpc.exe", "frpc executable path")
	fs.StringVar(&opts.workDir, "workdir", filepath.Join("data", "frpc"), "frpc work directory")
	fs.StringVar(&opts.server, "server", envOrDefault("FRP_PANEL_SERVER", localclient.DefaultRoomControlServerURL), "control server base URL")
	fs.StringVar(&opts.deviceName, "device-name", "", "this device name")
	fs.BoolVar(&opts.jsonOut, "json", false, "print JSON output")
	fs.BoolVar(&opts.detach, "detach", false, "return after starting frpc instead of keeping this client process in foreground")
}

func openDirectClient(opts directClientOptions) (*localclient.Service, error) {
	if strings.TrimSpace(opts.server) == "" {
		return nil, errors.New("--server or FRP_PANEL_SERVER is required")
	}
	options := []localclient.ServiceOption{
		localclient.WithServerBaseURL(opts.server),
	}
	if opts.detach {
		options = append(options, localclient.WithDetachedFrpcProcesses())
	}
	return localclient.OpenService(
		opts.data,
		opts.frpcPath,
		opts.workDir,
		options...,
	)
}

func runClientHost(args []string) error {
	fs := flag.NewFlagSet("client host", flag.ExitOnError)
	opts := directClientOptions{}
	addDirectClientFlags(fs, &opts)
	name := fs.String("name", "", "room name")
	localIP := fs.String("local-ip", "127.0.0.1", "local service IP")
	localPort := fs.Int("local-port", 0, "local service port")
	tunnelProtocol := fs.String("tunnel", string(localclient.RoomTunnelXTCP), "room tunnel protocol: xtcp or stcp")
	natHoleStunServer := fs.String("nat-hole-stun-server", "", "optional STUN server for XTCP NAT discovery")
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		*name = args[0]
		args = args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *localPort == 0 {
		return errors.New("--local-port is required")
	}
	svc, err := openDirectClient(opts)
	if err != nil {
		return err
	}
	rule, err := svc.CreateRoomHost(context.Background(), localclient.CreateRoomHostRequest{
		Name:              strings.TrimSpace(*name),
		DeviceName:        strings.TrimSpace(opts.deviceName),
		TunnelProtocol:    localclient.RoomTunnelProtocol(strings.TrimSpace(*tunnelProtocol)),
		NatHoleStunServer: strings.TrimSpace(*natHoleStunServer),
		LocalIP:           strings.TrimSpace(*localIP),
		LocalPort:         *localPort,
	})
	if err != nil {
		svc.Stop()
		return err
	}
	if opts.jsonOut {
		if err := printJSON(rule); err != nil {
			svc.Stop()
			return err
		}
	} else {
		fmt.Printf("room: %s\n", rule.RoomID)
		fmt.Printf("roomCode: %s\n", rule.RoomCode)
		fmt.Printf("serverName: %s\n", rule.ServerName)
		fmt.Printf("local: %s:%d\n", rule.LocalIP, rule.LocalPort)
		fmt.Printf("frps: %s:%d\n", rule.ServerAddr, rule.ServerPort)
	}
	return waitDirectClient(svc, opts.detach)
}

func runClientJoin(args []string) error {
	fs := flag.NewFlagSet("client join", flag.ExitOnError)
	opts := directClientOptions{}
	addDirectClientFlags(fs, &opts)
	name := fs.String("name", "", "local rule name")
	bindAddr := fs.String("bind-addr", "127.0.0.1", "local bind address")
	bindPort := fs.Int("bind-port", 0, "local bind port")
	tunnelProtocol := fs.String("tunnel", string(localclient.RoomTunnelXTCP), "room tunnel protocol: xtcp or stcp")
	natHoleStunServer := fs.String("nat-hole-stun-server", "", "optional STUN server for XTCP NAT discovery")
	roomCode := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		roomCode = args[0]
		args = args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(roomCode) == "" {
		return errors.New("client join requires a room code")
	}
	if *bindPort == 0 {
		return errors.New("--bind-port is required")
	}
	svc, err := openDirectClient(opts)
	if err != nil {
		return err
	}
	rule, err := svc.JoinRoom(context.Background(), localclient.JoinRoomRequest{
		RoomCode:          strings.TrimSpace(roomCode),
		Name:              strings.TrimSpace(*name),
		DeviceName:        strings.TrimSpace(opts.deviceName),
		TunnelProtocol:    localclient.RoomTunnelProtocol(strings.TrimSpace(*tunnelProtocol)),
		NatHoleStunServer: strings.TrimSpace(*natHoleStunServer),
		BindAddr:          strings.TrimSpace(*bindAddr),
		BindPort:          *bindPort,
	})
	if err != nil {
		svc.Stop()
		return err
	}
	if opts.jsonOut {
		if err := printJSON(rule); err != nil {
			svc.Stop()
			return err
		}
	} else {
		fmt.Printf("joined room: %s\n", rule.RoomID)
		fmt.Printf("serverName: %s\n", rule.ServerName)
		fmt.Printf("local bind: %s:%d\n", rule.BindAddr, rule.BindPort)
		fmt.Printf("frps: %s:%d\n", rule.ServerAddr, rule.ServerPort)
	}
	return waitDirectClient(svc, opts.detach)
}

func runClientRooms(args []string) error {
	fs := flag.NewFlagSet("client rooms", flag.ExitOnError)
	opts := directClientOptions{}
	addDirectClientFlags(fs, &opts)
	if err := fs.Parse(args); err != nil {
		return err
	}
	svc, err := localclient.OpenService(
		opts.data,
		opts.frpcPath,
		opts.workDir,
		localclient.WithServerBaseURL(opts.server),
	)
	if err != nil {
		return err
	}
	rules, err := svc.ListRoomRules()
	if err != nil {
		return err
	}
	if opts.jsonOut {
		raw, err := json.MarshalIndent(rules, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(raw))
		return nil
	}
	if len(rules) == 0 {
		fmt.Println("no local room rules")
		return nil
	}
	for _, rule := range rules {
		local := fmt.Sprintf("%s:%d", rule.LocalIP, rule.LocalPort)
		if rule.Role == localclient.RoomRoleVisitor {
			local = fmt.Sprintf("%s:%d", rule.BindAddr, rule.BindPort)
		}
		fmt.Printf("%s\t%s\t%s\t%s\t%t\n", rule.ID, rule.Role, rule.RoomID, local, rule.Enabled)
	}
	return nil
}

func waitDirectClient(svc *localclient.Service, detach bool) error {
	if detach {
		return nil
	}
	fmt.Fprintln(os.Stderr, "client is running; press Ctrl+C to stop")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	svc.Stop()
	return nil
}
