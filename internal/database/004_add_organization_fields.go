package database

func init() {
	RegisterMigration(&Migration{
		ID:          "004_add_organization_fields",
		Description: "Add organization identity fields to tenant_settings for legal pages (privacy, terms)",
		Up: map[string]string{
			"sqlite": `
ALTER TABLE tenant_settings ADD COLUMN organization_name TEXT;
ALTER TABLE tenant_settings ADD COLUMN organization_address TEXT;
ALTER TABLE tenant_settings ADD COLUMN organization_email TEXT;
ALTER TABLE tenant_settings ADD COLUMN organization_phone TEXT;
ALTER TABLE tenant_settings ADD COLUMN privacy_officer_email TEXT;
`,
			"postgres": `
ALTER TABLE tenant_settings ADD COLUMN IF NOT EXISTS organization_name TEXT;
ALTER TABLE tenant_settings ADD COLUMN IF NOT EXISTS organization_address TEXT;
ALTER TABLE tenant_settings ADD COLUMN IF NOT EXISTS organization_email TEXT;
ALTER TABLE tenant_settings ADD COLUMN IF NOT EXISTS organization_phone TEXT;
ALTER TABLE tenant_settings ADD COLUMN IF NOT EXISTS privacy_officer_email TEXT;
`,
		},
	})
}
