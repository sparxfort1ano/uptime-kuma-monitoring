// A generated module for Monitoring functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/monitoring/internal/dagger"
	"fmt"
	"strings"
	"time"
)

type Monitoring struct{}

// DeployApp composes up MariaDB, Uptime Kuma and mariadb-exporter linking them via docker compose.
// +cache="never"
func (m *Monitoring) DeployApp(
	ctx context.Context,
	dockerSock *dagger.Socket,
	// +defaultPath="/"
	src *dagger.Directory,
) (string, error) {
	return dag.Container().
		From("docker:cli").
		WithEnvVariable("CACHEBUSTER", time.Now().String()).
		WithUnixSocket("/var/run/docker.sock", dockerSock).
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{
			"docker", "compose",
			"-f", "docker-compose.app.yml",
			"--env-file", ".env.app",
			"up", "-d", "--wait",
		}).
		WithExec([]string{"sh", "/src/scripts/init-exporter.sh"}).
		Stdout(ctx)
}

// DeployMonitoring composes up Prometheus, Grafana, cAdvisor and node-exporter linking them via docker compose.
// +cache="never"
func (m *Monitoring) DeployMonitoring(
	ctx context.Context,
	dockerSock *dagger.Socket,
	// +defaultPath="/"
	src *dagger.Directory,
) (string, error) {
	return dag.Container().
		From("docker:cli").
		WithEnvVariable("CACHEBUSTER", time.Now().String()).
		WithUnixSocket("/var/run/docker.sock", dockerSock).
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{
			"docker", "compose",
			"-f", "docker-compose.monitoring.yml",
			"--env-file", ".env.monitoring",
			"up", "-d", "--build", "--wait",
		}).
		Stdout(ctx)
}

// ConfigureGrafana configures Data Sources and Dashboards via Grafana API.
// +cache="never"
func (m *Monitoring) ConfigureGrafana(
	ctx context.Context,
	dockerSock *dagger.Socket,
	// +defaultPath="/"
	src *dagger.Directory,
) (string, error) {
	return dag.Container().
		From("docker:cli").
		WithEnvVariable("CACHEBUSTER", time.Now().String()).
		WithUnixSocket("/var/run/docker.sock", dockerSock).
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{
			"sh", "-c",
			`set -a && . ./.env.monitoring && set +a && \
			tar -cf - grafana scripts | docker run -i --rm --network monitoring_net \
			-e GRAFANA_USER="$GRAFANA_USER" \
			-e GRAFANA_PASSWORD="$GRAFANA_PASSWORD" \
			alpine:3.20 sh -c "apk add --no-cache curl jq >/dev/null && tar -xf - -C / && sh /scripts/configure-grafana.sh"`,
		}).
		Stdout(ctx)
}

// DeployAll deploys the entire infrastructure and validates the status of all systems.
// +cache="never"
func (m *Monitoring) DeployAll(
	ctx context.Context,
	dockerSock *dagger.Socket,
	// +defaultPath="/"
	src *dagger.Directory,
) (string, error) {
	var log strings.Builder

	appOut, err := m.DeployApp(ctx, dockerSock, src)
	if err != nil {
		return "", fmt.Errorf("deploy-app failed: %w", err)
	}
	log.WriteString("=== 1. App Layer Deployed ===\n" + appOut + "\n")

	monOut, err := m.DeployMonitoring(ctx, dockerSock, src)
	if err != nil {
		return "", fmt.Errorf("deploy-monitoring failed: %w", err)
	}
	log.WriteString("=== 2. Monitoring Layer Deployed ===\n" + monOut + "\n")

	grafOut, err := m.ConfigureGrafana(ctx, dockerSock, src)
	if err != nil {
		return "", fmt.Errorf("configure-grafana failed: %w", err)
	}
	log.WriteString("=== 3. Grafana Configured ===\n" + grafOut + "\n")

	valOut, err := dag.Container().
		From("docker:cli").
		WithEnvVariable("CACHEBUSTER", time.Now().String()).
		WithUnixSocket("/var/run/docker.sock", dockerSock).
		WithExec([]string{
			"sh", "-c",
			`docker ps --format "table {{.Names}}\t{{.Status}}" && \
			docker run --rm --network monitoring_net curlimages/curl:8.10.1 -fsS http://prometheus:9090/-/healthy && \
			docker run --rm --network monitoring_net curlimages/curl:8.10.1 -fsS http://grafana:3000/api/health`,
		}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}
	log.WriteString("\n=== 4. System Validation Passed ===\n" + valOut + "\n")

	return log.String(), nil
}
