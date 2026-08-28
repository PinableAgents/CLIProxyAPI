package cliproxy

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	proxyconfig "github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	sdkaccess "github.com/router-for-me/CLIProxyAPI/v7/sdk/access"
)

func TestBuilderRegistersEphemeralAPIKeyWithAccessManager(t *testing.T) {
	accessManager := sdkaccess.NewManager()
	cfg := &proxyconfig.Config{AuthDir: filepath.Join(t.TempDir(), "auths")}
	service, errBuild := NewBuilder().
		WithConfig(cfg).
		WithConfigPath(filepath.Join(t.TempDir(), "config.yaml")).
		WithRequestAccessManager(accessManager).
		WithEphemeralAPIKey("runtime-secret").
		Build()
	if errBuild != nil {
		t.Fatalf("Build() error = %v", errBuild)
	}

	request, errRequest := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost/v1/models", nil)
	if errRequest != nil {
		t.Fatalf("create request: %v", errRequest)
	}
	request.Header.Set("Authorization", "Bearer runtime-secret")
	result, authErr := accessManager.Authenticate(context.Background(), request)
	if authErr != nil {
		t.Fatalf("Authenticate() error = %v", authErr)
	}
	if result.Provider != "cliproxyapi-ephemeral" {
		t.Fatalf("provider = %q, want cliproxyapi-ephemeral", result.Provider)
	}
	if result.Principal == "runtime-secret" {
		t.Fatal("authentication result exposed the ephemeral key")
	}

	service.syncPluginRuntimeConfigForConfig(context.Background(), cfg)
	result, authErr = accessManager.Authenticate(context.Background(), request)
	if authErr != nil {
		t.Fatalf("Authenticate() after plugin sync error = %v", authErr)
	}
	if result == nil {
		t.Fatal("Authenticate() after plugin sync returned no result")
	}
	if result.Provider != "cliproxyapi-ephemeral" {
		t.Fatalf("provider after plugin sync = %q, want cliproxyapi-ephemeral", result.Provider)
	}

	request.Header.Set("Authorization", "Bearer wrong-secret")
	if _, authErr = accessManager.Authenticate(context.Background(), request); !sdkaccess.IsAuthErrorCode(authErr, sdkaccess.AuthErrorCodeInvalidCredential) {
		t.Fatalf("wrong key error = %v, want invalid credential", authErr)
	}
}
