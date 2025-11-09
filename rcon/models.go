package rcon

import (
	"net"
	"sync"
	"time"
)

type RCONClient struct {
	IP       string
	Port     int
	Password string
	Timeout  time.Duration
	Conn     *net.UDPConn
	mu       sync.Mutex
}

type Player struct {
	ClientNum int
	Name      string
	Ping      any
	Score     int
	IP        string
	Port      int
	QPort     int
	GUID      string
	LastMsg   int
	Rate      int
}

type ServerStatus struct {
	Map         string
	Players     []Player
	Raw         []string
	RetrievedAt time.Time
}

type ServerInfo struct {
	NetFieldChk int64
	Protocol    int
	SessionMode int
	Hostname    string
	MapName     string
	IsInGame    bool
	MaxClients  int
	GameType    string
	HW          int
	Mod         bool
	Voice       bool
	SecKey      string
	SecID       string
	HostAddr    string
	RetrievedAt time.Time
}

type ServerStatusInfo struct {
	ComMaxClients            int
	GameType                 string
	RandomSeed               int
	GameName                 string
	MapName                  string
	PlaylistEnabled          bool
	PlaylistEntry            int
	Protocol                 int
	ScrTeamFFType            int
	ShortVersion             bool
	SvAllowAimAssist         bool
	SvAllowAnonymous         bool
	SvClientFpsLimit         int
	SvDisableClientConsole   bool
	SvHostname               string
	SvMaxClients             int
	SvMaxPing                int
	SvMinPing                int
	SvPatchDSR50             bool
	SvPrivateClients         int
	SvPrivateClientsForUsers int
	SvPure                   bool
	SvVoice                  bool
	PasswordEnabled          bool
	ModEnabled               bool
	RetrievedAt              time.Time
}

type commandSettings struct {
	retries        int
	readTimeout    time.Duration
	readExtension  time.Duration
	requireSuccess bool
}

type commandOption func(*commandSettings)
