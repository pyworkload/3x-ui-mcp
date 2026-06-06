package xui

import "encoding/json"

// Response is the standard 3x-ui API response envelope.
type Response struct {
	Success bool            `json:"success"`
	Msg     string          `json:"msg"`
	Obj     json.RawMessage `json:"obj"`
}

// Inbound represents a 3x-ui inbound connection.
type Inbound struct {
	ID             int    `json:"id"`
	Up             int64  `json:"up"`
	Down           int64  `json:"down"`
	Total          int64  `json:"total"`
	Remark         string `json:"remark"`
	Enable         bool   `json:"enable"`
	ExpiryTime     int64  `json:"expiryTime"`
	Listen         string `json:"listen"`
	Port           int    `json:"port"`
	Protocol       string `json:"protocol"`
	Settings       string `json:"settings"`
	StreamSettings string `json:"streamSettings"`
	Tag            string `json:"tag"`
	Sniffing       string `json:"sniffing"`
}

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
