package keyvault

import "strings"

// separator joins the "service" (everything before the last '/') and the key
// in a Key Vault secret name: internal "graph/DB_NAME" ↔ vault "graph--DB-NAME".
const separator = "--"

// ToKeyVaultName converts an internal parameter name to its Key Vault secret
// name: the last '/'-segment is the key (its underscores become hyphens,
// which Azure allows); everything before it is kept verbatim. Note that a
// multi-segment SSM-style path ("/dev/config") keeps its slashes, which Azure
// would reject — Key Vault configs use slash-free paths ("graph").
func ToKeyVaultName(internal string) string {
	service, key := "", internal
	if i := strings.LastIndex(internal, "/"); i >= 0 {
		service, key = internal[:i], internal[i+1:]
	}
	return service + separator + strings.ReplaceAll(key, "_", "-")
}

// FromKeyVaultName reverses ToKeyVaultName, splitting on the FIRST separator.
// It is lossy: every hyphen in the key becomes an underscore, so a key that
// legitimately contained '-' comes back different (Node parity).
func FromKeyVaultName(kvName string) string {
	i := strings.Index(kvName, separator)
	if i < 0 {
		return kvName
	}
	service, kvKey := kvName[:i], kvName[i+len(separator):]
	return service + "/" + strings.ReplaceAll(kvKey, "-", "_")
}
