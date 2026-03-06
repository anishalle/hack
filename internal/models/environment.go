package models

type ProviderType string

const (
	ProviderCloudRun      ProviderType = "cloud-run"
	ProviderComputeEngine ProviderType = "compute-engine"
	ProviderAppEngine     ProviderType = "app-engine"

	ProviderNeon        ProviderType = "neon"
	ProviderSupabase    ProviderType = "supabase"
	ProviderPlanetScale ProviderType = "planetscale"
	ProviderCloudSQL    ProviderType = "cloud-sql"

	ProviderSuperTokens  ProviderType = "supertokens"
	ProviderFirebaseAuth ProviderType = "firebase"
	ProviderSupabaseAuth ProviderType = "supabase-auth"

	ProviderGSM ProviderType = "google-secret-manager"
)

func ValidDeployProviders() []ProviderType {
	return []ProviderType{ProviderCloudRun, ProviderComputeEngine, ProviderAppEngine}
}

func ValidDBProviders() []ProviderType {
	return []ProviderType{ProviderNeon, ProviderSupabase, ProviderPlanetScale, ProviderCloudSQL}
}

func ValidAuthProviders() []ProviderType {
	return []ProviderType{ProviderSuperTokens, ProviderFirebaseAuth, ProviderSupabaseAuth}
}
