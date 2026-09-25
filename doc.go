// Package kodi is a JSON-RPC client for Kodi.
//
// Layout:
//
//	internal/rpc  shared, version-agnostic transport (Client, Call, RPCError)
//	v13           generated, typed client for Kodi API v13 (from schema/v13)
//	<root>        thin facade: defaults to the latest supported version plus
//	              version-agnostic negotiation (APIVersion, Negotiate)
//
// Regenerate versioned code with:
//
//	mise run regen
package kodi
