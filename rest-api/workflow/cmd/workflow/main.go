// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	sentryZerolog "github.com/getsentry/sentry-go/zerolog"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	zlogadapter "logur.dev/adapter/zerolog"
	"logur.dev/logur"

	tsdkClient "go.temporal.io/sdk/client"
	tsdkWorker "go.temporal.io/sdk/worker"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	cotel "github.com/NVIDIA/infra-controller/rest-api/common/pkg/otel"
	ctemporal "github.com/NVIDIA/infra-controller/rest-api/common/pkg/temporal"
	cdb "github.com/NVIDIA/infra-controller/rest-api/db/pkg/db"

	"github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/config"

	cwm "github.com/NVIDIA/infra-controller/rest-api/workflow/internal/metrics"
	cwfh "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/health"
	cwfn "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/namespace"

	sc "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/client/site"

	// Core/identity activities and workflows — owned by the workflow binary.
	// Domain activities (VPC, instance, machine, etc.) are owned by their
	// respective in-tree providers and run as workers in the API binary.
	userActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/user"
	userWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/user"

	siteActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/site"
	siteWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/site"

	tenantActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/tenant"
	tenantWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/tenant"
)

const (
	// ZerologMessageFieldName specifies the field name for log message
	ZerologMessageFieldName = "msg"
	// ZerologLevelFieldName specifies the field name for log level
	ZerologLevelFieldName = "type"
)

func main() {
	// Initialize logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.LevelFieldName = ZerologLevelFieldName
	zerolog.MessageFieldName = ZerologMessageFieldName

	if err := run(context.Background()); err != nil {
		log.Error().Err(err).Msg("workflow worker stopped with an error")
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg := config.NewConfig()
	defer cfg.Close()

	otelShutdown, err := cotel.Bootstrap(ctx, cfg.GetTracingEnabled(), cfg.GetTracingServiceName())
	if err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := otelShutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("failed to shut down tracing")
		}
	}()

	dbConfig := cfg.GetDBConfig()

	// Initialize DB connection
	dbSession, err := cdb.NewSession(ctx, dbConfig.Host, dbConfig.Port, dbConfig.Name, dbConfig.User, dbConfig.Password, "")
	if err != nil {
		return fmt.Errorf("failed to initialize DB session: %w", err)
	}
	defer dbSession.Close()

	log.Info().Msg("creating Temporal client")

	sentryDSN := os.Getenv("SENTRY_DSN")
	if sentryDSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              sentryDSN,
			Debug:            true,
			AttachStacktrace: true,
		})
		if err != nil {
			log.Error().Err(err).Msg("Sentry initialization failed")
		} else {
			defer sentry.Flush(2 * time.Second)

			sentryWriter, err := sentryZerolog.New(sentryZerolog.Config{
				ClientOptions: sentry.ClientOptions{Dsn: sentryDSN},
				Options: sentryZerolog.Options{
					Levels:          []zerolog.Level{zerolog.ErrorLevel, zerolog.FatalLevel, zerolog.PanicLevel},
					WithBreadcrumbs: true,
					FlushTimeout:    3 * time.Second,
				},
			})
			if err != nil {
				log.Error().Err(err).Msg("failed to create Sentry writer")
			} else {
				defer sentryWriter.Close()
				log.Logger = zerolog.New(zerolog.MultiLevelWriter(os.Stderr, sentryWriter))
			}
		}
	}

	tLogger := logur.LoggerToKV(zlogadapter.New(zerolog.New(os.Stderr)))
	var tc tsdkClient.Client

	tcfg, err := cfg.GetTemporalConfig()
	if err != nil {
		return fmt.Errorf("failed to get Temporal config: %w", err)
	}

	tOptions, err := ctemporal.ClientOptions(tcfg.GetHostPort(), tcfg.Namespace, tcfg.ClientTLSCfg, tLogger)
	if err != nil {
		return fmt.Errorf("failed to build Temporal client options: %w", err)
	}

	tc, err = tsdkClient.NewLazyClient(tOptions)
	if err != nil {
		return fmt.Errorf("failed to create Temporal client: %w", err)
	}
	defer tc.Close()

	w := tsdkWorker.New(tc, tcfg.Queue, tsdkWorker.Options{
		WorkflowPanicPolicy:              tsdkWorker.FailWorkflow,
		MaxConcurrentActivityTaskPollers: cfg.GetMaxConcurrentActivityPollers(),
		MaxConcurrentWorkflowTaskPollers: 10,
	})

	siteClientPool := sc.NewClientPool(tcfg)

	log.Info().Str("Temporal Namespace", tcfg.Namespace).Msg("registering core workflow and activities")

	// Core/identity workflows — these run on the main NICo task queue.
	// Domain workflows (networking, compute, site) are registered by their
	// respective providers running as workers in the API binary.
	if tcfg.Namespace == cwfn.CloudNamespace {
		w.RegisterWorkflow(userWorkflow.UpdateUserFromNGC)
		w.RegisterWorkflow(userWorkflow.UpdateUserFromNGCWithAuxiliaryID)
		w.RegisterWorkflow(siteWorkflow.DeleteSiteComponents)
		w.RegisterWorkflow(siteWorkflow.MonitorHealthForAllSites)
		w.RegisterWorkflow(siteWorkflow.MonitorTemporalCertExpirationForAllSites)
		w.RegisterWorkflow(siteWorkflow.MonitorSiteTemporalNamespaces)
	} else if tcfg.Namespace == cwfn.SiteNamespace {
		w.RegisterWorkflow(siteWorkflow.UpdateAgentCertExpiry)
		w.RegisterWorkflow(siteWorkflow.UpdateSiteConfigInventory)
		w.RegisterWorkflow(siteWorkflow.UpdateSiteConfigInventoryV2)
		w.RegisterWorkflow(tenantWorkflow.UpdateTenantInventory)
	}

	mconfig := cfg.GetMetricsConfig()

	var reg *prometheus.Registry
	var siteHealthMetrics *cwm.SiteHealthMetrics

	if mconfig.Enabled {
		reg = prometheus.NewRegistry()
		reg.MustRegister(collectors.NewGoCollector())

		cm := cwm.NewCoreMetrics(reg, mconfig.Namespace)
		cm.Info.With(prometheus.Labels{"version": "unknown", "namespace": tcfg.Namespace}).Set(1)

		siteHealthMetrics = cwm.NewSiteHealthMetrics(reg, mconfig.Namespace)

		if tcfg.Namespace == cwfn.SiteNamespace {
			inventoryMetricsManager := cwm.NewManageInventoryMetrics(reg, dbSession, mconfig.Namespace)
			w.RegisterActivity(inventoryMetricsManager)
		}
	}

	// Core/identity activities
	siteManager := siteActivity.NewManageSite(dbSession, siteClientPool, tc, cfg, siteHealthMetrics)
	w.RegisterActivity(&siteManager)

	tenantManager := tenantActivity.NewManageTenant(dbSession, siteClientPool)
	w.RegisterActivity(&tenantManager)

	if tcfg.Namespace == cwfn.CloudNamespace {
		userManager := userActivity.NewManageUser(dbSession, cfg)
		w.RegisterActivity(&userManager)
	}

	serveErrs := make(chan error, 2)
	serve := func(name, addr string) {
		go func() {
			log.Info().Msgf("starting %s server", name)
			if err := http.ListenAndServe(addr, nil); err != nil {
				serveErrs <- fmt.Errorf("%s server on %s: %w", name, addr, err)
			}
		}()
	}

	hconfig := cfg.GetHealthzConfig()
	if hconfig.Enabled {
		http.HandleFunc("/healthz", cwfh.StatusHandler)
		http.HandleFunc("/readyz", cwfh.StatusHandler)
		serve("health check API", hconfig.GetListenAddr())
	}

	if mconfig.Enabled {
		http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
		serve("Prometheus metrics", mconfig.GetListenAddr())
	}

	interrupt := make(chan interface{}, 1)
	go func() {
		select {
		case <-tsdkWorker.InterruptCh():
		case err := <-serveErrs:
			serveErrs <- err
		}
		interrupt <- struct{}{}
	}()

	log.Info().Str("Temporal Namespace", tcfg.Namespace).Msg("starting core Temporal worker")
	err = w.Run(interrupt)
	if err != nil {
		return fmt.Errorf("failed to run worker for Temporal namespace %s: %w", tcfg.Namespace, err)
	}
	select {
	case err := <-serveErrs:
		return err
	default:
	}

	if tcfg.Namespace == cwfn.CloudNamespace {
		_, err := siteWorkflow.ExecuteMonitorHealthForAllSitesWorkflow(ctx, tc)
		if err != nil {
			log.Error().Err(err).Msg("failed to trigger Site Health Monitor workflow")
		}

		_, err = siteWorkflow.ExecuteMonitorTemporalCertExpirationForAllSites(ctx, tc)
		if err != nil {
			log.Error().Err(err).Msg("failed to trigger Temporal Cert Expiration Monitor workflow")
		}

		_, err = siteWorkflow.ExecuteMonitorSiteTemporalNamespaces(ctx, tc)
		if err != nil {
			log.Error().Err(err).Msg("failed to trigger Monitor Site Temporal Namespaces workflow")
		}
	}

	return nil
}
