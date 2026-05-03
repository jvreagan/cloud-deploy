package credentials

import (
	"context"
	"testing"
)

func TestGetCredentials_Environment(t *testing.T) {
	m := &Manager{Source: "environment"}

	// Set AWS env vars
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	t.Setenv("AWS_SESSION_TOKEN", "token123")

	creds, err := m.GetCredentials(context.Background(), "aws")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AWS.AccessKeyID != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("got AccessKeyID=%q, want AKIAIOSFODNN7EXAMPLE", creds.AWS.AccessKeyID)
	}
	if creds.AWS.SecretAccessKey != "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" {
		t.Errorf("got SecretAccessKey=%q, want ...EXAMPLEKEY", creds.AWS.SecretAccessKey)
	}
	if creds.AWS.SessionToken != "token123" {
		t.Errorf("got SessionToken=%q, want token123", creds.AWS.SessionToken)
	}
}

func TestGetCredentials_UnknownSource(t *testing.T) {
	m := &Manager{Source: "magic"}
	_, err := m.GetCredentials(context.Background(), "aws")
	if err == nil {
		t.Fatal("expected error for unknown source")
	}
}

func TestGetFromEnvironment_AWS_Missing(t *testing.T) {
	m := &Manager{Source: "environment"}
	// Ensure env vars are unset
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")

	_, err := m.GetCredentials(context.Background(), "aws")
	if err == nil {
		t.Fatal("expected error when AWS env vars are missing")
	}
}

func TestGetFromEnvironment_GCP(t *testing.T) {
	m := &Manager{Source: "environment"}
	t.Setenv("GCP_PROJECT_ID", "my-project")
	t.Setenv("GCP_SERVICE_ACCOUNT_KEY", `{"type":"service_account"}`)

	creds, err := m.GetCredentials(context.Background(), "gcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.GCP.ProjectID != "my-project" {
		t.Errorf("got ProjectID=%q, want my-project", creds.GCP.ProjectID)
	}
	if creds.GCP.ServiceAccountKey != `{"type":"service_account"}` {
		t.Errorf("got ServiceAccountKey=%q", creds.GCP.ServiceAccountKey)
	}
}

func TestGetFromEnvironment_GCP_Missing(t *testing.T) {
	m := &Manager{Source: "environment"}
	t.Setenv("GCP_PROJECT_ID", "")
	t.Setenv("GCP_SERVICE_ACCOUNT_KEY", "")

	_, err := m.GetCredentials(context.Background(), "gcp")
	if err == nil {
		t.Fatal("expected error when GCP env vars are missing")
	}
}

func TestGetFromEnvironment_Azure(t *testing.T) {
	m := &Manager{Source: "environment"}
	t.Setenv("AZURE_TENANT_ID", "tenant-123")
	t.Setenv("AZURE_CLIENT_ID", "client-456")
	t.Setenv("AZURE_CLIENT_SECRET", "secret-789")
	t.Setenv("AZURE_SUBSCRIPTION_ID", "sub-000")

	creds, err := m.GetCredentials(context.Background(), "azure")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Azure.TenantID != "tenant-123" {
		t.Errorf("got TenantID=%q, want tenant-123", creds.Azure.TenantID)
	}
	if creds.Azure.ClientID != "client-456" {
		t.Errorf("got ClientID=%q, want client-456", creds.Azure.ClientID)
	}
	if creds.Azure.SubscriptionID != "sub-000" {
		t.Errorf("got SubscriptionID=%q, want sub-000", creds.Azure.SubscriptionID)
	}
}

func TestGetFromEnvironment_Azure_Missing(t *testing.T) {
	m := &Manager{Source: "environment"}
	t.Setenv("AZURE_TENANT_ID", "")
	t.Setenv("AZURE_CLIENT_ID", "")
	t.Setenv("AZURE_CLIENT_SECRET", "")

	_, err := m.GetCredentials(context.Background(), "azure")
	if err == nil {
		t.Fatal("expected error when Azure env vars are missing")
	}
}

func TestGetFromEnvironment_Cloudflare(t *testing.T) {
	m := &Manager{Source: "environment"}
	t.Setenv("CLOUDFLARE_API_TOKEN", "cf-token-abc")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "cf-account-123")

	creds, err := m.GetCredentials(context.Background(), "cloudflare")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Cloudflare.APIToken != "cf-token-abc" {
		t.Errorf("got APIToken=%q, want cf-token-abc", creds.Cloudflare.APIToken)
	}
	if creds.Cloudflare.AccountID != "cf-account-123" {
		t.Errorf("got AccountID=%q, want cf-account-123", creds.Cloudflare.AccountID)
	}
}

func TestGetFromEnvironment_Cloudflare_Missing(t *testing.T) {
	m := &Manager{Source: "environment"}
	t.Setenv("CLOUDFLARE_API_TOKEN", "")

	_, err := m.GetCredentials(context.Background(), "cloudflare")
	if err == nil {
		t.Fatal("expected error when Cloudflare env vars are missing")
	}
}

func TestGetFromEnvironment_UnknownProvider(t *testing.T) {
	m := &Manager{Source: "environment"}
	_, err := m.GetCredentials(context.Background(), "digitalocean")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestValidateCredentials_AWS_Valid(t *testing.T) {
	creds := &ProviderCredentials{}
	creds.AWS.AccessKeyID = "AKIAIOSFODNN7EXAMPLE"
	creds.AWS.SecretAccessKey = "secret"

	if err := ValidateCredentials(creds, "aws"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCredentials_AWS_Incomplete(t *testing.T) {
	creds := &ProviderCredentials{}
	creds.AWS.AccessKeyID = "AKIAIOSFODNN7EXAMPLE"
	// Missing SecretAccessKey

	if err := ValidateCredentials(creds, "aws"); err == nil {
		t.Fatal("expected error for incomplete AWS credentials")
	}
}

func TestValidateCredentials_GCP_Valid(t *testing.T) {
	creds := &ProviderCredentials{}
	creds.GCP.ProjectID = "my-project"
	creds.GCP.ServiceAccountKey = `{"type":"service_account"}`

	if err := ValidateCredentials(creds, "gcp"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCredentials_GCP_Incomplete(t *testing.T) {
	creds := &ProviderCredentials{}
	creds.GCP.ProjectID = "my-project"
	// Missing ServiceAccountKey

	if err := ValidateCredentials(creds, "gcp"); err == nil {
		t.Fatal("expected error for incomplete GCP credentials")
	}
}

func TestValidateCredentials_Azure_Valid(t *testing.T) {
	creds := &ProviderCredentials{}
	creds.Azure.TenantID = "tenant"
	creds.Azure.ClientID = "client"
	creds.Azure.ClientSecret = "secret"

	if err := ValidateCredentials(creds, "azure"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCredentials_Azure_Incomplete(t *testing.T) {
	creds := &ProviderCredentials{}
	creds.Azure.TenantID = "tenant"
	// Missing ClientID and ClientSecret

	if err := ValidateCredentials(creds, "azure"); err == nil {
		t.Fatal("expected error for incomplete Azure credentials")
	}
}

func TestValidateCredentials_Cloudflare_Valid(t *testing.T) {
	creds := &ProviderCredentials{}
	creds.Cloudflare.APIToken = "token"

	if err := ValidateCredentials(creds, "cloudflare"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCredentials_Cloudflare_Incomplete(t *testing.T) {
	creds := &ProviderCredentials{}
	// Missing APIToken

	if err := ValidateCredentials(creds, "cloudflare"); err == nil {
		t.Fatal("expected error for incomplete Cloudflare credentials")
	}
}

func TestValidateCredentials_UnknownProvider(t *testing.T) {
	creds := &ProviderCredentials{}
	if err := ValidateCredentials(creds, "unknown"); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
