package tron

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"

	"golang.org/x/crypto/ssh"
)

var (
	matchip    = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+`) //TODO: make correct
	filtername = regexp.MustCompile(`\W`)                  //non-words
)

type Server struct {
	port       int
	idPool     <-chan ID
	logf       func(format string, vars ...interface{})
	sshConfig  *ssh.ServerConfig
	newPlayers chan *Player
}

func NewServer(port int, idPool <-chan ID) (*Server, error) {
	config, err := generateConfig()
	if err != nil {
		return nil, err
	}
	s := &Server{
		port:       port,
		idPool:     idPool,
		logf:       log.New(os.Stdout, "server: ", 0).Printf,
		sshConfig:  config,
		newPlayers: make(chan *Player),
	}
	return s, nil
}

func (s *Server) start() {
	s.logf("up - join at")
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		ipv4 := matchip.FindString(a.String())
		if ipv4 != "" {
			s.logf("  ○ ssh %s -p %d", ipv4, s.port)
		}
	}
	// s.logf(" optionally provide a user@ name (player name)")

	// bind to provided port
	server, err := net.ListenTCP("tcp4", &net.TCPAddr{Port: s.port})
	if err != nil {
		log.Fatal(err)
	}
	// accept all tcp
	for {
		tcpConn, err := server.AcceptTCP()
		if err != nil {
			s.logf("accept error (%s)", err)
			continue
		}
		go s.handle(tcpConn)
	}
}

func (s *Server) handle(tcpConn *net.TCPConn) {

	// non-blocking pull off the id pool
	var id ID
	select {
	case id, _ = <-s.idPool:
	default:
	}

	// perform handshake
	sshConn, chans, globalReqs, err := ssh.NewServerConn(tcpConn, s.sshConfig)
	if err != nil {
		s.logf("failed to handshake (%s)", err)
		return
	}

	// global requests must be serviced - discard
	go func() {
		noop := func(arg interface{}) {}
		for req := range globalReqs {
			noop(req)
		}
	}()

	// get user and client info
	name := sshConn.User()
	// protect against XTR (cross terminal renderering) attacks
	name = filtername.ReplaceAllString(name, "")

	// trim name
	maxlen := sidebarWidth - 6
	if len(name) > maxlen {
		name = string([]rune(name)[:maxlen])
	}
	// default name
	if name == "" {
		s.logf("")
		name = fmt.Sprintf("player-%d", id)
	}

	// get the first channel
	c := <-chans

	// channel requests must be serviced - reject
	go func() {
		for c := range chans {
			c.Reject(ssh.Prohibited, "only 1 channel allowed")
		}
	}()

	// must be a 'session'
	if t := c.ChannelType(); t != "session" {
		c.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		return
	}

	conn, chanReqs, err := c.Accept()
	if err != nil {
		s.logf("could not accept channel (%s)", err)
		return
	}

	// show fullgame error
	if id == 0 {
		conn.Write([]byte("This game is full.\r\n"))
		tcpConn.Close()
		return
	}

	p := NewPlayer(id, name, conn)

	go func() {
		for r := range chanReqs {
			ok := false
			switch r.Type {
			case "shell":
				// We don't accept any commands (Payload),
				// only the default shell.
				if len(r.Payload) == 0 {
					ok = true
				}
			case "pty-req":
				// Responding 'ok' here will let the client
				// know we have a pty ready for input
				ok = true
				strlen := r.Payload[3]
				p.resizes <- parseDims(r.Payload[strlen+4:])
			case "window-change":
				p.resizes <- parseDims(r.Payload)
				continue // no response
			}
			r.Reply(ok, nil)

		}
	}()

	s.newPlayers <- p
}

// tron doesnt need security :) DONT DO THIS IRL
const privateKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAzNO5vZPpP7WgXA3Ck5NeCq85i1v2JCB5vM0udK+oWrCQpMdy
oKZlxC8z8n/mSsylm+2xEm+kAFxyvB9ae/Pr8Lh0czePw473Qx2v78E/HdouXn3w
xEHG12IoDUdC7Rt4faxNdfsebd/wWybHEV6vOEDDkxmppJ1y6Cbgx6a59X0wqW54
bTKy5D98iLMzSvWi6AUS3I/hP53f7mNK7cTPqHTdVOwICgCGHOI1hcDKwMafj590
+3H/F5ACYRl9Keuij09zsk+QkI+7HJN5HUtq9mjJ9Mw4vo9LzqIWTOWncEvX5b2f
99GOOlsBNh91L3PNwQdf1M++CM6F0HTv5p8ioQIDAQABAoIBACwruJF2dUWE8IkJ
ep2CmTQqp3kzIriVvEsH4G3Pd7ne+8JdNI4KdEXDfCteg5Y73bbrolT8eFyPkzqY
dFXou0fVL1+tarZcfVwe6dMFVIwmgftko2hfWvcVttduN7OUSf6oCqhXuC8vrNCr
YyCOz7CM3uA5F4llXuNLhwvnG5EhxHk/AVN0SUbJbfKD5DEpqFM33PuITAuIPuSi
Td2qa84WitZ12hBJqtZGngujE/bMZNaY0Lk6EM4L2p47+//z3raScQT2B+eF/LnR
Jn32YaI7np7Y4D7RbW6QZBB/sOkrvtX51tIHIQEYdn4zlfT8+tNeVo9jn0QM77Ky
FcY4a8ECgYEA5vF+P5MeSa+QsUVgK3HY4MuNNRKw4daIFJr/keYLyUwfPYQsdu5V
ZXfJPkQ/y1Xlgek6E/eiiaiJN91hZEkoF6fkXcORCCmjr19FfssC++arTKk/UPxT
y946yFscsZXosssCON7CskGLCiPMn7YwdwQiJ9uvKIxwB2ChfJ/trSkCgYEA4wzY
rp5Pz3lbXg6P7xqYibnIH847PW9GVMGNl6pXfhUkP3NqFD+Oc41S/wD/vv1SVSZ7
2ih56E7vctxtxc9b5wWcZfzRUbBWrSKwWO1ImqsBdFapxtoOynDL0uHnXaDrQCvW
UsI44d92gmO+MMYst9//I/sLRTrwYrrIvJOVALkCgYEAg0uqVeSDJKtOnKnveeOY
xHyVBCZjL5Hy/Zv9Tmo2KzQ+0o9xZBAttqk6XU8Z4bUs7QW2giGYY6DQmlUfCI/a
3lASMgh8TOK3b32/mc07HhFPNB9IovdBgLcQPlYmYwPyLqvh0Ik8sXE35gTiUa6X
sSJFdNmdpHTrQBZ82MhnrLkCgYB8wG06HKALhkmOd3/cR4eyfNKZry3bho1lOmf7
AkxKaYFeH6MUdwtlMCx/EmRy4ytev+NjLcQ1wVFNkhH6kwGTAQE7BFtagAJP5PRy
GAZBfV4yNv/X0642yx0ixJ7kUeuQecWr+S1Z5fdukzFICUs+yKOeeGxr4IN+K9Tp
0EkZeQKBgF58RcI6PZD7mayf0Z58gd+zb2WXL1rTGErYsbVgxkc/TFRaZYK0cb+n
V6WZNy6k5Amx54pv59U34sEiGqFb8xo9Q0o+jcdrirTJKvuJuGh5Hm/4jjRvu4O3
1Qr6yBnUTsDcXkDy8G0oenhDMceZEbIz+WOqmxKx7eGl0OxE0CNt
-----END RSA PRIVATE KEY-----
`

func generateConfig() (*ssh.ServerConfig, error) {
	// An SSH server is represented by a ServerConfig, which holds
	// certificate details and handles authentication of ServerConns.
	config := &ssh.ServerConfig{
		KeyboardInteractiveCallback: func(conn ssh.ConnMetadata, client ssh.KeyboardInteractiveChallenge) (*ssh.Permissions, error) {
			return nil, nil // no challenge, we just want username
		},
	}
	p, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, err
	}
	config.AddHostKey(p)
	return config, nil
}

// parseDims extracts two uint32s from the provided buffer.
func parseDims(b []byte) resize {
	if len(b) < 8 {
		return resize{
			width:  0,
			height: 0,
		}
	}
	w := binary.BigEndian.Uint32(b)
	h := binary.BigEndian.Uint32(b[4:])
	return resize{
		width:  w,
		height: h,
	}
}
