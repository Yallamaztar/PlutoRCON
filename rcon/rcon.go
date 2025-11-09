package rcon

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	readBufferSize       = 4096
	defaultReadTimeout   = time.Second
	defaultReadExtension = 350 * time.Millisecond
)

func New(ip, port, password string) (*RCONClient, error) {
	if password == "" {
		return nil, errors.New("RCON password cannot be empty")
	}

	portNum, err := strconv.Atoi(port)
	if err != nil {
		return nil, errors.New("invalid port number")
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, portNum))
	if err != nil {
		return nil, errors.New("failed to resolve UDP address")
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, errors.New("failed to establish UDP connection")
	}

	return &RCONClient{
		IP:       ip,
		Port:     portNum,
		Password: password,
		Timeout:  defaultReadTimeout,
		Conn:     conn,
		mu:       sync.Mutex{},
	}, nil
}

// Close the RCONClient UDP connection
func (rc *RCONClient) Close() error {
	return rc.Conn.Close()
}

// readResponse reads the response from the RCON
func (rc *RCONClient) readResponse(readTimeout, readExtension time.Duration) ([]string, error) {
	if readTimeout <= 0 {
		readTimeout = defaultReadTimeout
	}
	if readExtension < 0 {
		readExtension = 0
	}

	var buf bytes.Buffer
	deadline := time.Now().Add(readTimeout)

	for {
		if err := rc.Conn.SetReadDeadline(deadline); err != nil {
			return nil, err
		}
		tmp := make([]byte, readBufferSize)
		n, err := rc.Conn.Read(tmp)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				if buf.Len() == 0 {
					return nil, err
				}
				break
			}
			return nil, err
		}
		if n > 0 {
			buf.Write(tmp[:n])
			if readExtension > 0 {
				deadline = time.Now().Add(readExtension)
			}
		}
	}

	raw := normalizeRCON(buf.String())
	if raw == "" {
		return nil, nil
	}
	lines := splitNonEmptyLines(raw)
	return lines, nil
}
