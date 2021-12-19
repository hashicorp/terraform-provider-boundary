package provider

const (
	// IDKey is used for common SDK ID resource attribute
	IDKey = "id"
	// NameKey is used for common "name" resource attribute
	NameKey = "name"
	// DescriptionKey is used for common "description" resource attribute
	DescriptionKey = "description"
	// ScopeIdKey is used for common "scope_id" resource attribute
	ScopeIdKey = "scope_id"
	// TypeKey is used for common "type" resource attribute
	TypeKey = "type"
	// HostCatalogIdKey is used for common "host_catalog_id" resource attribute
	HostCatalogIdKey = "host_catalog_id"
	// AuthMethodIdKey is used for common "auth_method_id" resource attribute
	AuthMethodIdKey = "auth_method_id"
	// PluginIdKey is used for common "plugin_id" resource attribute
	PluginIdKey = "plugin_id"
	// PluginNameKey is used for common "plugin_name" resource attribute
	PluginNameKey = "plugin_name"
	// AttributesJsonKey is used for setting attributes and corresponds to the
	// API "attributes" key
	AttributesJsonKey = "attributes_json"
	// SecretsJsonKey is used for setting secrets and corresponds to the API
	// "secrets" key
	SecretsJsonKey = "secrets_json"
	// SecretsHmacKey is a read-only key used for ensuring we detect if secrets
	// have changed
	SecretsHmacKey = "secrets_hmac"
	// PreferredEndpointsKey is used for setting preferred endpoints
	PreferredEndpointsKey = "preferred_endpoints"
	// SyncIntervalSecondsKey is used for setting the interval seconds
	SyncIntervalSecondsKey = "sync_interval_seconds"
	// internalSecretsConfigHmacKey is used for storing an hmac of hmac from server +
	// config string
	internalSecretsConfigHmacKey = "internal_secrets_config_hmac"
	// internalHmacUsedForSecretsConfigHmacKey is used for storing the server-provided
	// hmac used when calculating the current value of secretsConfigHmacKey
	internalHmacUsedForSecretsConfigHmacKey = "internal_hmac_used_for_secrets_config_hmac"
	// internalNextUpdateUseSecretsKey is used by the hmac diffing function to indicate
	// that when update runs it should use the value currently found in
	// secrets_json
	internalNextUpdateUseSecretsKey = "internal_next_update_use_secrets_key"
)
