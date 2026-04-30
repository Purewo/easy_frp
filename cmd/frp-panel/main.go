package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"frp-ui-backend/internal/client"
	"frp-ui-backend/internal/control"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "server":
		err = runServer(os.Args[2:])
	case "client":
		err = runClient(os.Args[2:])
	case "ctl":
		err = runCtl(os.Args[2:])
	case "openapi":
		err = printOpenAPI()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: frp-panel <server|client|ctl|openapi> [flags]\n")
}

func runServer(args []string) error {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "HTTP listen address")
	data := fs.String("data", filepath.Join("data", "server.json"), "server JSON data file")
	frpsAddr := fs.String("frps-addr", envOrDefault("FRP_PANEL_FRPS_ADDR", ""), "public frps address for room clients")
	frpsPort := fs.Int("frps-port", envIntOrDefault("FRP_PANEL_FRPS_PORT", 7000), "public frps bind port for room clients")
	if err := fs.Parse(args); err != nil {
		return err
	}

	svc, err := control.OpenService(*data, control.WithFrpsEndpoint(*frpsAddr, *frpsPort))
	if err != nil {
		return err
	}
	return serve(*addr, control.NewHandler(svc))
}

func runClient(args []string) error {
	if len(args) > 0 && isClientDirectCommand(args[0]) {
		return runClientDirect(args)
	}

	fs := flag.NewFlagSet("client", flag.ExitOnError)
	defaultNode := client.DefaultNodeConfig()
	addr := fs.String("addr", "127.0.0.1:7410", "local HTTP listen address")
	data := fs.String("data", filepath.Join("data", "client.json"), "client JSON data file")
	frpcPath := fs.String("frpc", "frpc.exe", "frpc executable path")
	workDir := fs.String("workdir", filepath.Join("data", "frpc"), "frpc work directory")
	serverBaseURL := fs.String("server", envOrDefault("FRP_PANEL_SERVER", client.DefaultRoomControlServerURL), "control server base URL for room mode")
	nodeID := fs.String("node-id", envOrDefault("FRP_PANEL_NODE_ID", defaultNode.ID), "default frps node id")
	nodeName := fs.String("node-name", envOrDefault("FRP_PANEL_NODE_NAME", defaultNode.Name), "default frps node name")
	frpsHost := fs.String("frps-host", envOrDefault("FRP_PANEL_FRPS_HOST", defaultNode.ServerAddr), "default frps server address")
	frpsPort := fs.Int("frps-port", envIntOrDefault("FRP_PANEL_FRPS_PORT", defaultNode.FrpsPort), "default frps server port")
	frpsToken := fs.String("frps-token", os.Getenv("FRP_PANEL_FRPS_TOKEN"), "default frps auth token")
	webBaseDomain := fs.String("web-base-domain", envOrDefault("FRP_PANEL_WEB_BASE_DOMAIN", defaultNode.WebBaseDomain), "base domain for http web exposures")
	webScheme := fs.String("web-scheme", envOrDefault("FRP_PANEL_WEB_SCHEME", defaultNode.WebScheme), "public scheme for web exposures")
	vhostHTTPPort := fs.Int("vhost-http-port", envIntOrDefault("FRP_PANEL_VHOST_HTTP_PORT", defaultNode.VhostHTTPPort), "frps vhost HTTP port")
	if err := fs.Parse(args); err != nil {
		return err
	}

	defaultNode.ID = *nodeID
	defaultNode.Name = *nodeName
	defaultNode.ServerAddr = *frpsHost
	defaultNode.FrpsPort = *frpsPort
	defaultNode.AuthToken = *frpsToken
	defaultNode.WebBaseDomain = *webBaseDomain
	defaultNode.WebScheme = *webScheme
	defaultNode.VhostHTTPPort = *vhostHTTPPort
	svc, err := client.OpenService(*data, *frpcPath, *workDir, client.WithDefaultNode(defaultNode), client.WithServerBaseURL(*serverBaseURL))
	if err != nil {
		return err
	}
	return serve(*addr, client.NewHandler(svc))
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envIntOrDefault(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func printOpenAPI() error {
	data, err := os.ReadFile(filepath.Join("openapi", "openapi.yaml"))
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}

func serve(addr string, handler http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on http://%s", addr)
		errCh <- srv.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	case <-sigCh:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}
