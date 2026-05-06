// Package server ...
package server

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/service"
	"github.com/ognev-dev/goplease/game/match"
	"github.com/ognev-dev/goplease/game/ws"
	"github.com/ognev-dev/goplease/server/endpoint"
	"github.com/ognev-dev/goplease/server/handler"
	"github.com/ognev-dev/goplease/server/middleware"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/acme/autocert"
)

// RWTimeout defines server's Read&Write timeout in seconds.
const RWTimeout = 10 * time.Second

// New creates new server.
func New(s *service.Service, t trace.Tracer) *http.Server {
	conf := app.Config().Server

	registerOAuthProviders()

	h := handler.New(s, t)
	mw := middleware.New(s, t)
	r := endpoint.NewRouter(mw, h)

	// game endpoints
	mm := match.New()
	hub := ws.NewHub()
	gs := ws.NewGameServer(hub, mm)

	go hub.Run()
	go gs.Run()

	r.GET("/goplease/", hub.ServeWS)

	r.HandleAssets()

	// Middlewares that is common to "web" and "api" endpoint groups
	common := r.Use(
		mw.Tracing,
		mw.Recovery,
		mw.Logging,
		mw.ResolveUserFromCookie,
	)

	// API endpoints
	api := common.Group(conf.APIBasePath)
	api.Use(mw.ServeJSON)

	api.PublicAPIEndpoints()
	api.Use(mw.UserAuth)
	api.ProtectedAPIEndpoints()

	var tlsConf *tls.Config
	if conf.AutocertHosts != "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			log.Fatal(err)
		}
		cacheDir = filepath.Join(cacheDir, "autocert")
		err = os.MkdirAll(cacheDir, 0700) //nolint:mnd
		if err != nil {
			log.Fatal(err)
		}

		hosts := strings.Split(conf.AutocertHosts, ",")
		for i, host := range hosts {
			hosts[i] = strings.TrimSpace(host)
		}
		am := &autocert.Manager{
			Cache:      autocert.DirCache(cacheDir),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(hosts...),
		}

		tlsConf = am.TLSConfig()
		go func() {
			pongTLS := &http.Server{
				Addr:         ":80",
				Handler:      am.HTTPHandler(nil),
				ReadTimeout:  RWTimeout,
				WriteTimeout: RWTimeout,
			}
			err := pongTLS.ListenAndServe()
			if err != nil {
				log.Println("autocert ListenAndServe: ", err.Error())
			}
		}()
	}

	return &http.Server{
		Addr:         net.JoinHostPort(conf.Host, conf.Port),
		Handler:      r,
		TLSConfig:    tlsConf,
		ReadTimeout:  RWTimeout,
		WriteTimeout: RWTimeout,
	}
}

func registerOAuthProviders() {
	c := app.Config()

	callbackURL := func(providerName string) string {
		return fmt.Sprintf("%sauth/%s/callback/", c.Server.Addr, providerName)
	}

	goth.UseProviders(
		google.New(c.GoogleOAuth.ClientID, c.GoogleOAuth.ClientSecret, callbackURL("google")),
		github.New(c.GithubOAuth.ClientID, c.GithubOAuth.ClientSecret, callbackURL("github")),
	)
}
