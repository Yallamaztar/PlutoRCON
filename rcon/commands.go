package rcon

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Send RCON command with optional arguments and settings
func (rc *RCONClient) SendCommand(cmd string, args *string, opts ...commandOption) ([]string, error) {
	if rc.Conn == nil {
		return nil, fmt.Errorf("RCON connection is not established")
	}

	s := commandSettings{
		retries:        3,
		readTimeout:    rc.timeoutOrDefault(),
		readExtension:  defaultReadExtension,
		requireSuccess: false,
	}

	for _, opt := range opts {
		opt(&s)
	}

	var payload string
	if args != nil && strings.TrimSpace(*args) != "" {
		payload = fmt.Sprintf("rcon %s %s %s", rc.Password, strings.TrimSpace(cmd), strings.TrimSpace(*args))
	} else {
		payload = fmt.Sprintf("rcon %s %s", rc.Password, strings.TrimSpace(cmd))
	}
	packet := append([]byte{0xFF, 0xFF, 0xFF, 0xFF}, []byte(payload)...)
	packet = append(packet, '\n')

	rc.mu.Lock()
	defer rc.mu.Unlock()

	var lerr error
	for i := 0; i <= s.retries; i++ {
		if _, err := rc.Conn.Write(packet); err != nil {
			lerr = err
			if i < s.retries {
				time.Sleep(time.Duration(i+1) * 150 * time.Millisecond)
			}
			continue
		}

		res, err := rc.readResponse(s.readTimeout, s.readExtension)
		if len(res) > 0 {
			return res, nil
		}

		if err == nil {
			if !s.requireSuccess {
				return res, nil
			}
		} else {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				if !s.requireSuccess {
					return nil, nil
				}
				lerr = err
			} else {
				return nil, err
			}
		}
		if i < s.retries {
			time.Sleep(time.Duration(i+1) * 150 * time.Millisecond)
		}
	}

	if s.requireSuccess {
		if lerr != nil {
			return nil, lerr
		}
		return nil, fmt.Errorf("no response received for command %q", cmd)
	}

	return nil, lerr
}

// Server Status
func (rc *RCONClient) Status() (*ServerStatus, error) {
	res, err := rc.SendCommand("status", nil, requireResponse(), withReadExtension(1*time.Second))
	if err != nil {
		return nil, err
	}

	status := &ServerStatus{Raw: res, RetrievedAt: time.Now()}

	for _, line := range res {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "map:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				status.Map = strings.TrimSpace(parts[1])
			}
			break
		}
	}

	start := -1
	headerRx := regexp.MustCompile(`(?i)^num\s+score\s+ping`)
	for i, line := range res {
		if headerRx.MatchString(strings.TrimSpace(line)) {
			start = i + 1
			break
		}
	}
	if start == -1 {
		start = 0
	}

	lines := res[start:]
	pattern := regexp.MustCompile(
		`(?P<num>\d+)\s+` +
			`(?P<score>-?\d+)\s+` +
			`(?P<bot>\w+)?\s*` +
			`(?P<ping>\d+|LOAD)\s+` +
			`(?P<guid>[0-9a-fA-F]+)\s+` +
			`(?P<name>.+?)\s+` +
			`(?P<lastmsg>\d+)\s+` +
			`(?P<ipport>\S+)\s+` +
			`(?P<qport>\d+)\s+` +
			`(?P<rate>\d+)`,
	)

	var players []Player
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if c := line[0]; c < '0' || c > '9' {
			continue
		}
		match := pattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		group := func(name string) string {
			for i, n := range pattern.SubexpNames() {
				if n == name {
					return match[i]
				}
			}
			return ""
		}

		ipport := group("ipport")
		ip, portStr, _ := strings.Cut(ipport, ":")
		port, _ := strconv.Atoi(portStr)

		pingStr := group("ping")
		var ping any = pingStr
		if pingStr != "LOAD" {
			if val, err := strconv.Atoi(pingStr); err == nil {
				ping = val
			}
		}

		player := Player{
			ClientNum: atoi(group("num")),
			Name:      group("name"),
			Ping:      ping,
			Score:     atoi(group("score")),
			IP:        ip,
			Port:      port,
			QPort:     atoi(group("qport")),
			GUID:      group("guid"),
			LastMsg:   atoi(group("lastmsg")),
			Rate:      atoi(group("rate")),
		}

		players = append(players, player)
	}

	status.Players = players
	return status, nil
}

// Say a message to all players
func (rc *RCONClient) Say(message string) error {
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}
	_, err := rc.SendCommand("say", &message)
	return err
}

// Tell a player a message
func (rc *RCONClient) Tell(clientNum int, message string) error {
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	arg := fmt.Sprintf("%d [^5Gambling^7] %s", clientNum, message)
	_, err := rc.SendCommand("tell", &arg)
	return err
}

// Kick a player with reason
func (rc *RCONClient) Kick(player, reason string) error {
	if player == "" || reason == "" {
		return fmt.Errorf("player and reason cannot be empty")
	}

	cmd := fmt.Sprintf("%s '%s'", player, reason)
	_, err := rc.SendCommand("clientkick_for_reason", &cmd)
	return err
}

// Set dvar value
func (rc *RCONClient) SetDvar(dvar, value string) error {
	if dvar == "" || value == "" {
		return fmt.Errorf("dvar and value cannot be empty")
	}

	if strings.ContainsAny(value, " \t\"") {
		value = fmt.Sprintf("\"%s\"", strings.ReplaceAll(value, "\"", "\\\""))
	}

	cmd := fmt.Sprintf("%s %s", dvar, value)
	_, err := rc.SendCommand("set", &cmd)
	return err
}

// Get dvar value
func (rc *RCONClient) GetDvar(dvar string) (string, error) {
	if dvar == "" {
		return "", fmt.Errorf("dvar cannot be empty")
	}

	name := regexp.QuoteMeta(strings.TrimSpace(dvar))
	rx1 := regexp.MustCompile(fmt.Sprintf(`(?i)^%s\s+is:\s+"?(?P<val>.*?)"?(?:\s|$)`, name))
	rx2 := regexp.MustCompile(fmt.Sprintf(`(?i)^%s\s*[:=]\s*"?(?P<val>.*?)"?$`, name))

	const maxAttempts = 3
	var lastClean string
	for attempt := 0; attempt < maxAttempts; attempt++ {
		res, err := rc.SendCommand(dvar, nil, requireResponse())
		if err != nil {
			return "", err
		}

		for _, line := range res {
			clean := strings.TrimSpace(stripColorCodes(line))
			if clean == "" {
				continue
			}
			if m := rx1.FindStringSubmatch(clean); m != nil {
				for i, n := range rx1.SubexpNames() {
					if n == "val" {
						return stripColorCodes(m[i]), nil
					}
				}
			}
			if m := rx2.FindStringSubmatch(clean); m != nil {
				for i, n := range rx2.SubexpNames() {
					if n == "val" {
						return stripColorCodes(m[i]), nil
					}
				}
			}
			if !strings.Contains(strings.ToLower(clean), "sv_iw4madmin_in") {
				if lastClean == "" {
					lastClean = clean
				}
			}
		}
		needRetry := false
		for _, line := range res {
			if strings.Contains(strings.ToLower(line), "sv_iw4madmin_in") {
				needRetry = true
				break
			}
		}
		if !needRetry {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 150 * time.Millisecond)
	}

	if lastClean != "" {
		return lastClean, nil
	}
	return "", fmt.Errorf("empty dvar response for %q", dvar)
}

// Get Server Info
func (rc *RCONClient) GetInfo() (*ServerInfo, error) {
	if rc.Conn == nil {
		return nil, fmt.Errorf("RCON connection is not established")
	}

	packet := append([]byte{0xFF, 0xFF, 0xFF, 0xFF}, []byte("getinfo")...)
	packet = append(packet, '\n')

	rc.mu.Lock()
	defer rc.mu.Unlock()

	if _, err := rc.Conn.Write(packet); err != nil {
		return nil, err
	}
	lines, err := rc.readResponse(rc.timeoutOrDefault(), defaultReadExtension)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty infoResponse")
	}

	var dataLine string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.EqualFold(t, "inforesponse") {
			continue
		}
		if strings.HasPrefix(t, "\\") || strings.Contains(t, "\\") {
			if dataLine == "" {
				dataLine = t
			} else {
				dataLine += t
			}
		}
	}
	if dataLine == "" {
		dataLine = strings.TrimSpace(lines[len(lines)-1])
	}

	parts := strings.Split(dataLine, "\\")
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	kv := map[string]string{}
	for i := 0; i < len(parts)-1; i += 2 {
		key := strings.TrimSpace(parts[i])
		val := strings.TrimSpace(parts[i+1])
		if key == "" {
			continue
		}
		kv[key] = stripColorCodes(val)
	}

	info := &ServerInfo{RetrievedAt: time.Now()}

	atoint64Safe := func(k string) int64 {
		v, ok := kv[k]
		if !ok {
			return 0
		}
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	}

	info.NetFieldChk = atoint64Safe("netfieldchk")
	info.Protocol = atoi("protocol")
	info.SessionMode = atoi("sessionmode")
	info.Hostname = kv["hostname"]
	info.MapName = kv["mapname"]
	info.IsInGame = boolSafe("isInGame")
	info.MaxClients = atoi("com_maxclients")
	info.GameType = kv["gametype"]
	info.HW = atoi("hw")
	info.Mod = boolSafe("mod")
	info.Voice = boolSafe("voice")
	info.SecKey = kv["seckey"]
	info.SecID = kv["secid"]
	info.HostAddr = kv["hostaddr"]

	return info, nil
}

// Get Server Status
func (rc *RCONClient) GetStatus() (*ServerStatusInfo, error) {
	if rc.Conn == nil {
		return nil, fmt.Errorf("RCON connection is not established")
	}

	packet := append([]byte{0xFF, 0xFF, 0xFF, 0xFF}, []byte("getstatus")...)
	packet = append(packet, '\n')

	rc.mu.Lock()
	defer rc.mu.Unlock()

	if _, err := rc.Conn.Write(packet); err != nil {
		return nil, err
	}

	lines, err := rc.readResponse(rc.timeoutOrDefault(), defaultReadExtension)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty statusResponse")
	}

	var dataLine string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.EqualFold(t, "statusresponse") {
			continue
		}
		if strings.HasPrefix(t, "\\") || strings.Contains(t, "\\") {
			if dataLine == "" {
				dataLine = t
			} else {
				dataLine += t
			}
		}
	}
	if dataLine == "" {
		dataLine = strings.TrimSpace(lines[len(lines)-1])
	}

	parts := strings.Split(dataLine, "\\")
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	kv := map[string]string{}
	for i := 0; i < len(parts)-1; i += 2 {
		k := strings.TrimSpace(parts[i])
		v := strings.TrimSpace(parts[i+1])
		if k == "" {
			continue
		}
		kv[k] = stripColorCodes(v)
	}

	info := &ServerStatusInfo{RetrievedAt: time.Now()}

	info.ComMaxClients = atoi("com_maxclients")
	info.GameType = kv["g_gametype"]
	info.RandomSeed = atoi("g_randomSeed")
	info.GameName = kv["gamename"]
	info.MapName = kv["mapname"]
	info.PlaylistEnabled = boolSafe("playlist_enabled")
	info.PlaylistEntry = atoi("playlist_entry")
	info.Protocol = atoi("protocol")
	info.ScrTeamFFType = atoi("scr_team_fftype")
	info.ShortVersion = boolSafe("shortversion")
	info.SvAllowAimAssist = boolSafe("sv_allowAimAssist")
	info.SvAllowAnonymous = boolSafe("sv_allowAnonymous")
	info.SvClientFpsLimit = atoi("sv_clientFpsLimit")
	info.SvDisableClientConsole = boolSafe("sv_disableClientConsole")
	info.SvHostname = kv["sv_hostname"]
	info.SvMaxClients = atoi("sv_maxclients")
	info.SvMaxPing = atoi("sv_maxPing")
	info.SvMinPing = atoi("sv_minPing")
	info.SvPatchDSR50 = boolSafe("sv_patch_dsr50")
	info.SvPrivateClients = atoi("sv_privateClients")

	if v, ok := kv["sv_privateClientsForClients"]; ok {
		info.SvPrivateClientsForUsers, _ = strconv.Atoi(v)
	} else {
		info.SvPrivateClientsForUsers = atoi("sv_privateClientsForUsers")
	}
	info.SvPure = boolSafe("sv_pure")
	info.SvVoice = boolSafe("sv_voice")
	info.PasswordEnabled = boolSafe("pswrd")
	info.ModEnabled = boolSafe("mod")

	return info, nil
}
