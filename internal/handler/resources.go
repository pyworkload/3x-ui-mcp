package handler

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Resources are the lazy half of this server: a tool call spends context on its
// result whether or not the caller needed all of it, while a resource costs one
// line in resources/list and is only fetched when something asks for the URI.
// The heavy tools return a summary plus a resource_link to the full payload.
const (
	docsInboundSettingsURI = "xui://docs/inbound-settings"
	docsClientFieldsURI    = "xui://docs/client-fields"
	xrayConfigURI          = "xui://xray/config"
	xrayTemplateURI        = "xui://xray/template"
	inboundsURI            = "xui://inbounds"
	inboundLinksURI        = "xui://inbounds/links"
	clientsExportURI       = "xui://clients/export"
)

// resourceHandler holds the XUI client for resource reads.
type resourceHandler struct {
	client *xui.Client
}

func registerResources(s *server.MCPServer, client *xui.Client) {
	h := &resourceHandler{client: client}

	s.AddResource(mcp.NewResource(docsInboundSettingsURI, "Inbound settings reference",
		mcp.WithResourceDescription("The settings, streamSettings and sniffing JSON that create_inbound and update_inbound take, with a worked example per protocol."),
		mcp.WithMIMEType("text/markdown"),
	), staticResource(docsInboundSettingsURI, "text/markdown", inboundSettingsDoc))

	s.AddResource(mcp.NewResource(docsClientFieldsURI, "Client fields reference",
		mcp.WithResourceDescription("What each client field means and the units it uses — quotas, expiry, IP limits, subscription and Telegram identifiers."),
		mcp.WithMIMEType("text/markdown"),
	), staticResource(docsClientFieldsURI, "text/markdown", clientFieldsDoc))

	s.AddResource(mcp.NewResource(xrayConfigURI, "Running Xray configuration",
		mcp.WithResourceDescription("The full Xray config the core is running right now."),
		mcp.WithMIMEType("application/json"),
	), h.jsonResource(xrayConfigURI, func(ctx context.Context) (*xui.Response, error) {
		return h.client.GetXrayConfig(ctx)
	}))

	s.AddResource(mcp.NewResource(xrayTemplateURI, "Xray template",
		mcp.WithResourceDescription("The saved Xray template: routing rules, balancers, outbounds and DNS, before the panel merges the inbounds in."),
		mcp.WithMIMEType("application/json"),
	), h.jsonResource(xrayTemplateURI, func(ctx context.Context) (*xui.Response, error) {
		return h.client.GetXrayTemplate(ctx)
	}))

	s.AddResource(mcp.NewResource(inboundsURI, "Inbounds",
		mcp.WithResourceDescription("Every inbound with its settings, stream settings and per-client stats."),
		mcp.WithMIMEType("application/json"),
	), h.jsonResource(inboundsURI, func(ctx context.Context) (*xui.Response, error) {
		return h.client.ListInbounds(ctx)
	}))

	s.AddResource(mcp.NewResource(inboundLinksURI, "Connection links",
		mcp.WithResourceDescription("Connection URLs for every client across every inbound, rendered by the panel's subscription engine."),
		mcp.WithMIMEType("application/json"),
	), h.jsonResource(inboundLinksURI, func(ctx context.Context) (*xui.Response, error) {
		return h.client.GetAllInboundLinks(ctx)
	}))

	s.AddResource(mcp.NewResource(clientsExportURI, "Client export",
		mcp.WithResourceDescription("Every client as a {client, inboundIds} array — the shape import_clients accepts. Includes credentials."),
		mcp.WithMIMEType("application/json"),
	), h.jsonResource(clientsExportURI, func(ctx context.Context) (*xui.Response, error) {
		return h.client.ExportClients(ctx)
	}))

	s.AddResourceTemplate(mcp.NewResourceTemplate("xui://inbound/{id}", "Inbound by ID",
		mcp.WithTemplateDescription("One inbound with its full settings, addressed by numeric ID."),
		mcp.WithTemplateMIMEType("application/json"),
	), h.readInbound)

	s.AddResourceTemplate(mcp.NewResourceTemplate("xui://client/{email}", "Client by email",
		mcp.WithTemplateDescription("One client record with its inbound attachments, addressed by email."),
		mcp.WithTemplateMIMEType("application/json"),
	), h.readClient)
}

// staticResource serves text that never changes, so it costs no panel call.
func staticResource(uri, mimeType, text string) server.ResourceHandlerFunc {
	return func(context.Context, mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return []mcp.ResourceContents{mcp.TextResourceContents{
			URI:      uri,
			MIMEType: mimeType,
			Text:     text,
		}}, nil
	}
}

// jsonResource turns one panel call into a resource read. Unlike a tool, a
// resource has no isError channel, so an API-level failure becomes a Go error.
func (h *resourceHandler) jsonResource(uri string, call func(context.Context) (*xui.Response, error)) server.ResourceHandlerFunc {
	return func(ctx context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		text, err := resourceText(call(ctx))
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "application/json",
			Text:     text,
		}}, nil
	}
}

func (h *resourceHandler) readInbound(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	raw := strings.TrimPrefix(req.Params.URI, "xui://inbound/")
	id, err := strconv.Atoi(raw)
	if err != nil {
		return nil, fmt.Errorf("inbound id must be numeric, got %q", raw)
	}
	text, err := resourceText(h.client.GetInbound(ctx, id))
	if err != nil {
		return nil, err
	}
	return []mcp.ResourceContents{mcp.TextResourceContents{
		URI:      req.Params.URI,
		MIMEType: "application/json",
		Text:     text,
	}}, nil
}

func (h *resourceHandler) readClient(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	raw := strings.TrimPrefix(req.Params.URI, "xui://client/")
	email, err := url.PathUnescape(raw)
	if err != nil {
		return nil, fmt.Errorf("client email is not a valid URI segment: %w", err)
	}
	if email == "" {
		return nil, fmt.Errorf("client email is required, e.g. xui://client/alice")
	}
	text, err := resourceText(h.client.GetClient(ctx, email))
	if err != nil {
		return nil, err
	}
	return []mcp.ResourceContents{mcp.TextResourceContents{
		URI:      req.Params.URI,
		MIMEType: "application/json",
		Text:     text,
	}}, nil
}

// resourceText is toResult's counterpart for reads that have no tool result to
// carry the message.
func resourceText(resp *xui.Response, err error) (string, error) {
	if err != nil {
		return "", err
	}
	if !resp.Success {
		return "", fmt.Errorf("API error: %s", resp.Msg)
	}
	return formatResponse(resp), nil
}

const inboundSettingsDoc = "# Inbound settings\n\n" + `
` + "`create_inbound` and `update_inbound` take three JSON strings. This is what goes in them.\n" + `
## settings (protocol-specific)

VLESS — clients plus a decryption mode:

    {"clients":[{"id":"<uuid>","flow":"xtls-rprx-vision","email":"alice","limitIp":0,
                 "totalGB":0,"expiryTime":0,"enable":true,"tgId":"","subId":""}],
     "decryption":"none","fallbacks":[]}

VMess — same clients, no decryption field:

    {"clients":[{"id":"<uuid>","email":"alice","limitIp":0,"totalGB":0,
                 "expiryTime":0,"enable":true}]}

Trojan — password instead of id:

    {"clients":[{"password":"<secret>","email":"alice","limitIp":0,"totalGB":0,
                 "expiryTime":0,"enable":true}],"fallbacks":[]}

Shadowsocks — a per-inbound method and password, plus per-client passwords:

    {"method":"2022-blake3-aes-256-gcm","password":"<server-key>","network":"tcp,udp",
     "clients":[{"password":"<client-key>","email":"alice"}]}

## streamSettings (transport)

Plain TCP:

    {"network":"tcp","security":"none","tcpSettings":{"header":{"type":"none"}}}

REALITY over TCP — generate the keypair with ` + "`generate_key`" + ` (type x25519)
and pick a target with ` + "`scan_reality_target`" + `:

    {"network":"tcp","security":"reality","realitySettings":{
       "dest":"example.com:443","serverNames":["example.com"],
       "privateKey":"<from generate_key>","shortIds":["0123abcd"]}}

WebSocket behind a proxy:

    {"network":"ws","security":"none","wsSettings":{"path":"/ws","host":""}}

## sniffing

    {"enabled":true,"destOverride":["http","tls","quic","fakedns"],
     "metadataOnly":false,"routeOnly":false}

Sniffing is what makes domain-based routing rules work: without it the router
only sees IP addresses.
`

const clientFieldsDoc = "# Client fields\n" + `
Clients are keyed by ` + "`email`" + `, which is an arbitrary label and need not be an
address. One client can be attached to several inbounds at once.

| Field | Meaning |
|---|---|
| ` + "`email`" + ` | Unique label and the key every client tool takes |
| ` + "`id`" + ` / ` + "`password`" + ` | UUID for VMess/VLESS, secret for Trojan/Shadowsocks. Generated by the panel when omitted |
| ` + "`flow`" + ` | XTLS flow, usually ` + "`xtls-rprx-vision`" + ` or empty |
| ` + "`limitIp`" + ` | Concurrent IP limit; 0 means unlimited |
| ` + "`totalGB`" + ` | Traffic quota **in bytes** despite the name; 0 means unlimited. Tool parameters named ` + "`total_gb`" + ` take GB and are converted |
| ` + "`expiryTime`" + ` | Unix **milliseconds**; 0 means never. A negative value is a countdown that starts on first connect |
| ` + "`enable`" + ` | Whether the client may connect |
| ` + "`tgId`" + ` | Telegram user ID for bot notifications |
| ` + "`subId`" + ` | Subscription ID; clients sharing one subId share a subscription URL |
| ` + "`group`" + ` | Group label, managed by the client-group tools |
| ` + "`reset`" + ` | Traffic reset cycle in days; 0 disables the cycle |
| ` + "`comment`" + ` | Free-text note |

Updates are read-modify-write: ` + "`update_client`" + ` reads the current record and
overlays only the fields you pass, so omitting a field keeps it rather than
clearing it.
`
