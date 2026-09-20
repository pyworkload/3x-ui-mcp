package xui

import "encoding/json"

// Response is the standard 3x-ui API response envelope.
type Response struct {
	Success bool            `json:"success"`
	Msg     string          `json:"msg"`
	Obj     json.RawMessage `json:"obj"`
}

// There is deliberately no Inbound struct. The panel's inbound shape drifts
// between releases — v3.3.1 started emitting settings/streamSettings/sniffing
// as nested objects instead of JSON-encoded strings, and v3.8.x added
// shareAddr, shareAddrStrategy, subSortIndex, disableFlow, trafficReset,
// trafficResetDay and the node columns. Modelling it here made the update
// path both fail to parse the new shape and silently drop every field the
// struct did not know, so inbound bodies are carried as map[string]any and
// only the fields the panel computes per request are removed (see
// inboundRuntimeFields in internal/handler/inbound.go).

// ClientConfig represents a 3x-ui client (model.Client). It is sent both inside
// a ClientCreatePayload (clients/add) and as the bare body of clients/update/:email.
type ClientConfig struct {
	ID         string `json:"id,omitempty"`       // UUID (VMess/VLESS)
	Security   string `json:"security,omitempty"` // Cipher/security method (e.g. Shadowsocks)
	Password   string `json:"password,omitempty"` // Password (Trojan/Shadowsocks)
	Flow       string `json:"flow,omitempty"`     // XTLS flow (VLESS)
	Auth       string `json:"auth,omitempty"`     // Auth password (Hysteria)
	Email      string `json:"email"`
	LimitIP    int    `json:"limitIp"`
	TotalGB    int64  `json:"totalGB"`
	ExpiryTime int64  `json:"expiryTime"`
	Enable     bool   `json:"enable"`
	TgID       int64  `json:"tgId"`
	SubID      string `json:"subId"`
	Group      string `json:"group,omitempty"` // Logical grouping label
	Comment    string `json:"comment,omitempty"`
	Reset      int    `json:"reset"`

	// Fields below are optional and only sent when set, so a create that says
	// nothing about them leaves the panel to its own defaults. On the panel
	// side they belong to model.Client, except LimitHwid, which clients/add
	// reads from inside the client object while clients/update reads it from
	// the top level of the body (see the handler's update path).
	ResetDay        int            `json:"resetDay,omitempty"`     // Calendar renewal day 1-31, 0 = interval mode
	ResetMax        int            `json:"resetMax,omitempty"`     // Max auto-renews, 0 = unlimited
	TrafficReset    string         `json:"trafficReset,omitempty"` // never|hourly|daily|weekly|monthly
	TrafficResetDay int            `json:"trafficResetDay,omitempty"`
	LimitHwid       int            `json:"limitHwid,omitempty"` // Max registered devices, 0 = unlimited
	Reverse         *ClientReverse `json:"reverse,omitempty"`   // VLESS simple reverse proxy
	Secret          string         `json:"secret,omitempty"`    // MTProto per-client secret
	AdTag           string         `json:"adTag,omitempty"`     // MTProto ad tag, 32 hex chars
	PrivateKey      string         `json:"privateKey,omitempty"`
	PublicKey       string         `json:"publicKey,omitempty"`
	PreSharedKey    string         `json:"preSharedKey,omitempty"`
	AllowedIPs      []string       `json:"allowedIPs,omitempty"`
	KeepAlive       *int           `json:"keepAlive,omitempty"`      // WireGuard PersistentKeepalive seconds; 0 sends none
	ForwardedPorts  string         `json:"forwardedPorts,omitempty"` // AmneziaWG spec, e.g. "80,443,8000-8100"
}

// ClientReverse is the VLESS simple reverse proxy setting on a client.
type ClientReverse struct {
	Tag string `json:"tag"`
}

// ClientCreatePayload is the body for clients/add and each element of clients/bulkCreate.
type ClientCreatePayload struct {
	Client     ClientConfig `json:"client"`
	InboundIds []int        `json:"inboundIds"`
}

// InboundSettings wraps the clients array within inbound settings JSON.
type InboundSettings struct {
	Clients []ClientConfig `json:"clients"`
}
