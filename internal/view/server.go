package view

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/lagz0ne/sft/internal/render"
	"github.com/lagz0ne/sft/internal/show"
	"github.com/lagz0ne/sft/internal/store"
)

type Options struct {
	Port   int
	WebDir string
}

type Server struct {
	store      *store.Store
	opts       Options
	wsPort     int // internal NATS WS port (127.0.0.1 only)
	natsServer *natsserver.Server
	natsConn   *nats.Conn
	httpServer *http.Server
}

func NewServer(s *store.Store, opts Options) *Server {
	if opts.Port == 0 {
		opts.Port = 51741
	}
	if opts.WebDir == "" {
		opts.WebDir = findWebDir()
	}
	return &Server{store: s, opts: opts, wsPort: opts.Port + 1}
}

func (srv *Server) Start() error {
	// 1. Embedded NATS server — no external TCP, WS on loopback only
	natsOpts := &natsserver.Options{
		DontListen: true,
		Websocket: natsserver.WebsocketOpts{
			Host:  "127.0.0.1",
			Port:  srv.wsPort,
			NoTLS: true,
		},
		NoSigs: true,
		NoLog:  true,
	}
	ns, err := natsserver.NewServer(natsOpts)
	if err != nil {
		return fmt.Errorf("nats server: %w", err)
	}
	ns.Start()
	if !ns.ReadyForConnections(10 * time.Second) {
		return fmt.Errorf("nats server not ready")
	}
	srv.natsServer = ns

	// 2. In-process NATS client
	nc, err := nats.Connect("", nats.InProcessServer(ns))
	if err != nil {
		return fmt.Errorf("nats connect: %w", err)
	}
	srv.natsConn = nc

	// 3. Subscribe handlers
	if _, err := nc.Subscribe("sft.spec", srv.handleSpec); err != nil {
		return fmt.Errorf("subscribe sft.spec: %w", err)
	}
	if _, err := nc.Subscribe("sft.render", srv.handleRender); err != nil {
		return fmt.Errorf("subscribe sft.render: %w", err)
	}

	// 4. HTTP server — single port serves everything
	mux := http.NewServeMux()
	mux.HandleFunc("GET /nats", srv.proxyNatsWS) // WS upgrade → internal NATS
	mux.HandleFunc("GET /a/{entity}/{name}", srv.handleAttachment)
	mux.Handle("GET /deps/", http.StripPrefix("/deps/", http.FileServer(http.Dir(filepath.Join(srv.opts.WebDir, "node_modules")))))
	mux.Handle("GET /src/", http.StripPrefix("/src/", http.FileServer(http.Dir(filepath.Join(srv.opts.WebDir, "src")))))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(srv.opts.WebDir, "index.html"))
	})

	srv.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", srv.opts.Port),
		Handler: mux,
	}

	ln, err := net.Listen("tcp", srv.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("http listen: %w", err)
	}
	go srv.httpServer.Serve(ln)

	log.Printf("sft view → http://localhost:%d", srv.opts.Port)

	// 5. Watch for database changes — poll PRAGMA data_version
	stopWatch := make(chan struct{})
	go srv.watchChanges(stopWatch)

	// Block on signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	close(stopWatch)
	log.Println("shutting down...")
	return srv.Stop()
}

func (srv *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if srv.httpServer != nil {
		srv.httpServer.Shutdown(ctx)
	}
	if srv.natsConn != nil {
		srv.natsConn.Close()
	}
	if srv.natsServer != nil {
		srv.natsServer.Shutdown()
	}
	return nil
}

// proxyNatsWS proxies a WebSocket upgrade from the single HTTP port to the
// internal NATS WS listener on loopback. The browser sees only one port.
func (srv *Server) proxyNatsWS(w http.ResponseWriter, r *http.Request) {
	// Dial internal NATS WS
	backendConn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", srv.wsPort), 5*time.Second)
	if err != nil {
		http.Error(w, "nats unavailable", http.StatusBadGateway)
		return
	}

	// Rewrite path to / (NATS WS expects root)
	r.URL.Path = "/"
	r.RequestURI = "/"

	// Forward the original upgrade request to backend
	if err := r.Write(backendConn); err != nil {
		backendConn.Close()
		http.Error(w, "forward failed", http.StatusBadGateway)
		return
	}

	// Hijack client connection
	hj, ok := w.(http.Hijacker)
	if !ok {
		backendConn.Close()
		http.Error(w, "hijack unsupported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hj.Hijack()
	if err != nil {
		backendConn.Close()
		return
	}

	// Bidirectional pipe
	go func() {
		io.Copy(backendConn, clientConn)
		backendConn.Close()
	}()
	io.Copy(clientConn, backendConn)
	clientConn.Close()
}

func (srv *Server) handleSpec(msg *nats.Msg) {
	spec, err := show.Load(srv.store.DB, srv.store)
	if err != nil {
		msg.Respond(errJSON(err))
		return
	}
	data, _ := json.Marshal(spec)
	msg.Respond(data)
}

func (srv *Server) handleRender(msg *nats.Msg) {
	spec, err := show.Load(srv.store.DB, srv.store)
	if err != nil {
		msg.Respond(errJSON(err))
		return
	}
	jr := render.FromSFT(spec)
	render.Hydrate(jr, func(name string) *render.CompDef {
		comp := srv.store.GetComponentByName(name)
		if comp == nil {
			return nil
		}
		return &render.CompDef{
			Component: comp.Component,
			Props:     comp.Props,
			OnActions: comp.OnActions,
			Visible:   comp.Visible,
		}
	})
	data, _ := json.Marshal(jr)
	msg.Respond(data)
}

func (srv *Server) handleAttachment(w http.ResponseWriter, r *http.Request) {
	entity := r.PathValue("entity")
	name := r.PathValue("name")
	data, err := srv.store.ReadAttachment(entity, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType(data))
	w.Write(data)
}

// watchChanges polls PRAGMA data_version to detect external DB modifications
// (e.g. CLI commands in another terminal) and publishes sft.changes so the
// browser can re-fetch.
func (srv *Server) watchChanges(stop chan struct{}) {
	var lastVersion int64
	_ = srv.store.DB.QueryRow("PRAGMA data_version").Scan(&lastVersion)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			var version int64
			if err := srv.store.DB.QueryRow("PRAGMA data_version").Scan(&version); err != nil {
				continue
			}
			if version != lastVersion {
				lastVersion = version
				srv.natsConn.Publish("sft.changes", nil)
			}
		}
	}
}

func errJSON(err error) []byte {
	data, _ := json.Marshal(map[string]string{"error": err.Error()})
	return data
}

// findWebDir locates the web/ directory relative to the binary.
func findWebDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "web"
	}
	dir := filepath.Dir(exe)
	webDir := filepath.Join(dir, "web")
	if info, err := os.Stat(webDir); err == nil && info.IsDir() {
		return webDir
	}
	return "web"
}
