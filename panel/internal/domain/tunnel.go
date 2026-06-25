package domain

type TunnelProvider string

const (
	TunnelProviderPlayit   TunnelProvider = "playit"
	TunnelProviderDuckDNS  TunnelProvider = "duckdns"
)

type Tunnel struct {
	ServerID string
	Provider TunnelProvider
	Address  string // e.g. abc.ply.gg:25565
	Active   bool
}
