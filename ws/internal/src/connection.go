package src

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"strconv"
	"strings"

	"github.com/epicoon/lxgo/kernel/conv"
	"github.com/epicoon/lxgo/ws"
)

const (
	CloseCodeNormal               = 1000
	CloseCodeAccessError          = 1002
	CloseCodeProtocolError        = 1003
	CloseCodeUnknownData          = 1004
	CloseCodeLargeFrame           = 1005
	CloseCodeSocketError          = 1006
	CloseCodeWrongEncoding        = 1007
	CloseCodePolicyViolation      = 1008
	CloseCodeRequestLimitExceeded = 1009
)

/** @interface ws.IConnection */
type Connection struct {
	server         ws.IWSServer
	conn           net.Conn
	id             string
	ip             string
	status         int
	sharedData     map[string]any
	channels       map[string]map[string]any
	isReadyToClose bool
}

var _ ws.IConnection = (*Connection)(nil)

/** @constructor */
func NewConnection(s ws.IWSServer, conn net.Conn) ws.IConnection {
	addr := conn.RemoteAddr().String()
	addrParts := strings.Split(addr, ":")
	return &Connection{
		server:         s,
		conn:           conn,
		id:             RandHash(),
		ip:             addrParts[0],
		status:         ws.ConnStatusCreated,
		sharedData:     map[string]any{},
		channels:       map[string]map[string]any{},
		isReadyToClose: false,
	}
}

func (c *Connection) SetID(ID string) {
	c.id = ID
}

func (c *Connection) SetStatus(stat int) {
	if stat > ws.ConnStatusClosed {
		c.server.LifecycleError("wrong connection status: %d", stat)
		return
	}
	c.server.LifecycleLog("connection '%s' status changed from %d to %d", c.ID(), c.status, stat)
	c.status = stat
}

func (c *Connection) SetChannels(keys map[string]map[string]any) {
	c.channels = keys
}

func (c *Connection) ID() string {
	return c.id
}

func (c *Connection) IP() string {
	return c.ip
}

func (c *Connection) Status() int {
	return c.status
}

func (c *Connection) SharedData() map[string]any {
	return c.sharedData
}

func (c *Connection) SharedDataForChannel(ch ws.IChannel) map[string]any {
	if _, exists := c.channels[ch.Key()]; !exists {
		return c.sharedData
	}
	temp := map[string]any{}
	maps.Copy(temp, c.sharedData)
	maps.Copy(temp, c.channels[ch.Key()])
	return temp
}

func (c *Connection) Channels() map[string]map[string]any {
	return c.channels
}

func (c *Connection) Handle() {
	defer c.Close()

	// IP limit
	if !c.server.Connections().CheckIPLimit(c) {
		c.server.LifecycleLog("limit for IP: %v", c.IP())
		return
	}

	// Handshake
	reader, err := c.handshake()
	if err != nil {
		c.server.LifecycleError("handshake failed: %v", err)
		return
	}

	// Successful handshake
	c.server.LifecycleLog("handshake done for %s", c.id)
	c.SetStatus(ws.ConnStatusConnecting)
	hsResp := map[string]any{"id": c.ID()}
	if c.server.ReconnectionAllowed() {
		hsResp["reconnectionAllowed"] = true
	}
	c.server.Connections().Add(c)
	if err := c.Send(hsResp, "text", false); err != nil {
		c.server.LifecycleError("handshake response send error for %s: %v", c.id, err)
	}

	// Waiting for messages
	for {
		opcode, payload, fin, err := hybi10Decode(reader)
		if err != nil {
			c.server.LifecycleError("readFrame error: %v", err)
			return
		}

		if !c.server.Connections().CheckRequestLimit(c) {
			c.server.LifecycleLog("break connection %s: %s", c.ID(), "rate_limit_exceeded")
			c.Break("rate_limit_exceeded")
			return
		}

		switch opcode {
		case 0x1: // text
			// payload — []byte with text (utf-8)
			c.server.LifecycleLog("text (fin=%v): %s", fin, string(payload))
			c.handleTextMessage(payload)
		case 0x2:
			c.server.LifecycleLog("binary (len=%d)", len(payload))
			//TODO
		case 0x8:
			// close: optional 2-byte code + reason
			c.server.LifecycleLog("close frame: %v", payload)
			return
		case 0x9:
			// ping -> pong
			if err := c.Send(payload, "pong", false); err != nil {
				c.server.LifecycleError("pong send error for %s: %v", c.id, err)
			}
		case 0xA:
			// pong
		default:
			c.server.LifecycleError("unknown opcode %x", opcode)
		}
	}
}

func (c *Connection) Send(payload any, typ string, masked bool) error {
	var data []byte
	switch v := payload.(type) {
	case nil:
		data = []byte{}
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
		data = b
	}

	var opcode byte
	switch typ {
	case "text":
		opcode = 0x1
	case "binary":
		opcode = 0x2
	case "close":
		opcode = 0x8
	case "ping":
		opcode = 0x9
	case "pong":
		opcode = 0xA
	default:
		return fmt.Errorf("unsupported frame type: %s", typ)
	}

	frame, err := hybi10Encode(data, opcode, masked)
	if err != nil {
		return err
	}

	// Write frame to the TCP-connection
	n, err := c.conn.Write(frame)
	if err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	if n != len(frame) {
		return io.ErrShortWrite
	}
	return nil
}

func (c *Connection) Close() {
	if c.Status() == ws.ConnStatusClosed {
		return
	}

	if c.Status() == ws.ConnStatusDisconnected {
		c.SetStatus(ws.ConnStatusClosed)
		c.LeaveAllChannels()
		return
	}

	if c.server.Connections().Has(c.id) {
		if c.isReadyToClose {
			c.server.Connections().RemoveImmediate(c)
			c.SetStatus(ws.ConnStatusClosed)
		} else {
			c.server.Connections().MarkDisconnected(c)
			c.SetStatus(ws.ConnStatusDisconnected)
		}
	}

	c.LeaveAllChannels()

	if err := c.conn.Close(); err != nil {
		c.server.LifecycleError("close error for %s: %v", c.id, err)
	}
	c.conn = nil
	c.server.LifecycleLog("closed %s", c.id)
}

func (c *Connection) Break(msg string) {
	if err := c.Send(map[string]any{"error": msg}, "close", false); err != nil {
		c.server.LifecycleError("break send error for %s: %v", c.id, err)
	}
	c.Close()
}

func (c *Connection) IsChannelMate(ch ws.IChannel) bool {
	return ch.Has(c)
}

func (c *Connection) EnterChannel(ch ws.IChannel, message map[string]any) bool {
	res := ch.Enter(c, message)
	if !res {
		return false
	}

	c.channels[ch.Key()] = map[string]any{}
	if raw, ok := message["sharedData"]; ok {
		if m, ok := raw.(map[string]any); ok {
			c.channels[ch.Key()] = m
		}
	}
	return true
}

func (c *Connection) LeaveChannel(ch ws.IChannel) {
	if !c.IsChannelMate(ch) {
		return
	}
	delete(c.channels, ch.Key())
	ch.Leave(c)
}

func (c *Connection) LeaveAllChannels() {
	for key := range c.channels {
		ch := c.server.Channels().Get(key)
		ch.Leave(c)
	}
	c.channels = map[string]map[string]any{}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func (c *Connection) handshake() (*bufio.Reader, error) {
	reader := bufio.NewReader(c.conn)

	var lines []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && len(line) == 0 {
				return nil, errors.New("client closed before handshake")
			}
			if err != io.EOF {
				return nil, fmt.Errorf("reading handshake: %w", err)
			}
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		lines = append(lines, line)

		// Skip too long header
		if len(lines) > 1000 {
			return nil, errors.New("handshake header too large")
		}
	}

	if len(lines) == 0 {
		return nil, errors.New("empty handshake")
	}

	// Check first line: "GET /path HTTP/1.1"
	if !strings.HasPrefix(lines[0], "GET ") || !strings.Contains(lines[0], "HTTP/1.1") {
		return nil, fmt.Errorf("invalid request line: %q", lines[0])
	}

	// Build headers map
	headers := make(map[string]string)
	for _, l := range lines[1:] {
		if i := strings.Index(l, ":"); i != -1 {
			k := strings.TrimSpace(l[:i])
			v := strings.TrimSpace(l[i+1:])
			k = strings.ToLower(k)
			headers[k] = v
		}
	}

	secKey, ok := headers["sec-websocket-key"]
	if !ok || secKey == "" {
		return nil, errors.New("missing Sec-WebSocket-Key")
	}

	verStr := headers["sec-websocket-version"]
	if verStr == "" {
		// TODO param from config?
		verStr = "13"
	}
	ver, err := strconv.Atoi(verStr)
	if err != nil || ver < 6 {
		return nil, fmt.Errorf("unsupported websocket version: %s", verStr)
	}

	// Compute Sec-WebSocket-Accept
	accept := computeAcceptKey(secKey)

	// Prepare response
	resp := "HTTP/1.1 101 Switching Protocols\r\n"
	resp += "Upgrade: websocket\r\n"
	resp += "Connection: Upgrade\r\n"
	resp += "Sec-WebSocket-Accept: " + accept + "\r\n"

	// If Subprotocol used: header "sec-websocket-protocol" need to be returned
	if proto, ok := headers["sec-websocket-protocol"]; ok && proto != "" {
		//TODO - now return clients header
		resp += "Sec-WebSocket-Protocol: " + proto + "\r\n"
	}
	resp += "\r\n"

	_, err = c.conn.Write([]byte(resp))
	if err != nil {
		return nil, fmt.Errorf("write handshake response: %w", err)
	}

	return reader, nil
}

func computeAcceptKey(secKey string) string {
	//TODO guid to config params?
	const guid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.Sum([]byte(secKey + guid))
	return base64.StdEncoding.EncodeToString(h[:])
}

func (c *Connection) handleTextMessage(payload []byte) {
	// Parse JSON
	var msg any
	if err := json.Unmarshal(payload, &msg); err != nil {
		c.server.LifecycleError("invalid JSON payload: %v", err)
		c.Break("invalid_json")
		return
	}

	// Process object
	if m, ok := msg.(map[string]any); ok {
		// Multi request
		if multi, _ := m["__multi__"].(bool); multi {
			if list, ok := m["__list__"].([]any); ok {
				for _, item := range list {
					if imap, ok := item.(map[string]any); ok {
						c.processMessageMap(imap)
					}
				}
				return
			}
		}

		// Single request
		c.processMessageMap(m)
		return
	}
}

func (c *Connection) processMessageMap(message map[string]any) {
	if _, ok := message["__lxws_action__"]; ok {
		c.processAction(message)
		return
	}
	if _, ok := message["__lxws_request__"]; ok {
		c.processRequest(message)
		return
	}
	if _, ok := message["__lxws_channel__"]; ok {
		c.processChannelMsg(message)
		return
	}
}

func (c *Connection) processAction(message map[string]any) {
	// Action: message
	// 	["__lxws_action__"] string ("connect", "reconnect", "addSharedData", "close", "break")
	//  ["oldConnectionId"] string
	// 	["auth"] object
	// 	["shared"] object

	action, ok := message["__lxws_action__"].(string)
	if !ok {
		c.server.LifecycleError("invalid __lxws_action__ format")
		return
	}

	switch action {
	case "connect":
		//TODO auth
		c.extractSharedData(message)
		c.connect()

	case "reconnect":
		//TODO auth
		c.extractSharedData(message)

		oldID, ok := message["oldConnectionId"].(string)
		if !ok {
			c.server.LifecycleError("invalid oldConnectionId format")
			c.connect()
			return
		}

		if c.server.Connections().Reconnect(c, oldID) {
			c.reconnect()
		} else {
			c.connect()
		}

	case "close":
		c.isReadyToClose = true
		if err := c.Send(map[string]any{"__lxws_action__": "close"}, "text", false); err != nil {
			c.server.LifecycleError("close action send error for %s: %v", c.id, err)
		}

	case "break":
		if err := c.Send(map[string]any{"__lxws_action__": "break"}, "text", false); err != nil {
			c.server.LifecycleError("break action send error for %s: %v", c.id, err)
		}
	}
}

func (c *Connection) extractSharedData(message map[string]any) {
	shared, ok := message["shared"].(map[string]any)
	if ok {
		c.sharedData = shared
	}
}

func (c *Connection) connect() {
	data := map[string]any{"__lxws_action__": "connect"}

	if c.server.DefaultChannelKey() != "" {
		defaultChannel := c.server.Channels().Get(c.server.DefaultChannelKey())
		c.EnterChannel(defaultChannel, map[string]any{})
		data["channel"] = map[string]any{
			"key":         defaultChannel.Key(),
			"data":        defaultChannel.SharedData(),
			"connections": defaultChannel.MatesData(),
		}
	}

	c.SetStatus(ws.ConnStatusActive)
	if err := c.Send(data, "text", false); err != nil {
		c.server.LifecycleError("connect send error for %s: %v", c.id, err)
	}
}

func (c *Connection) reconnect() {
	data := map[string]any{
		"__lxws_action__": "reconnect",
		"idRestored":      c.ID(),
	}

	chs := []map[string]any{}
	for key := range c.Channels() {
		ch := c.server.Channels().Get(key)
		chs = append(chs, map[string]any{
			"key":         ch.Key(),
			"data":        ch.SharedData(),
			"connections": ch.MatesData(),
		})
	}
	if len(chs) > 0 {
		data["channels"] = chs
	}

	c.SetStatus(ws.ConnStatusActive)
	if err := c.Send(data, "text", false); err != nil {
		c.server.LifecycleError("reconnect send error for %s: %v", c.id, err)
	}
}

func (c *Connection) processRequest(message map[string]any) {
	// message
	// 	["__lxws_request__"]
	// 		["route"] string
	// 		["key"]   string
	// 	["__data__"] any

	reqRaw := message["__lxws_request__"]
	req, ok := reqRaw.(map[string]any)
	if !ok {
		c.server.LifecycleError("invalid __lxws_request__ format")
		return
	}
	route, err := conv.GetMapItem[string](req, "route")
	if err != nil {
		c.server.LifecycleError("can not get request route: %v", err)
		return
	}
	reqKey, err := conv.GetMapItem[string](req, "key")
	if err != nil {
		c.server.LifecycleError("can not get request key: %v", err)
		return
	}

	rawParams := message["__data__"]
	var params map[string]any
	if rawParams == nil {
		params = make(map[string]any)
	} else {
		params, ok = rawParams.(map[string]any)
		if !ok {
			c.server.LifecycleError("invalid request params format: route - %s, params - %v", route, rawParams)
			return
		}
	}

	resp := c.server.Router().Handle(route, params)

	msg := map[string]any{
		"__lxws_response__": true,
		"key":               reqKey,
		"code":              resp.Code(),
		"headers":           resp.Headers(),
		"body":              resp.Data(),
	}
	if err := c.Send(msg, "text", false); err != nil {
		c.server.LifecycleError("response send error for %s: %v", c.id, err)
	}
}

type chMsgOptions struct {
	Type string `dict:"__lxws_channel__"`
	Data any    `dict:"__data__"`
	Meta struct {
		Channel   string   `dict:"channel"`
		Receivers []string `dict:"receivers"`
		Re        bool     `dict:"returnToSender"`
		Private   bool     `dict:"private"`
		Event     string   `dict:"event"`
	} `dict:"__metaData__"`
}

func (c *Connection) processChannelMsg(message map[string]any) {
	// message
	// 	["__lxws_channel__"] string ("message", "event", "sharedData")
	// 	["__data__"] any
	//  ["__metaData__"]
	//  	["channel"] string
	//		["receivers"] []string|nil
	//		["returnToSender"] bool
	//		["private"] bool
	//		["event"] string|nil

	op := &chMsgOptions{}
	conv.MapToStruct(message, op)

	if _, exists := c.channels[op.Meta.Channel]; !exists {
		return
	}
	ch := c.server.Channels().Get(op.Meta.Channel)
	if ch == nil {
		return
	}

	switch op.Type {
	case "message":
		msg := NewChannelMessage(ch)
		msg.SetSender(c.ID())
		if len(op.Meta.Receivers) > 0 {
			msg.SetReceiverIds(op.Meta.Receivers)
		}
		msg.ReturnToSender(op.Meta.Re).
			SetPrivate(op.Meta.Private).
			SetData(op.Data)
		ws.SendMessage(msg)

	case "sharedData":
		c.channels[ch.Key()] = map[string]any{}
		if data, ok := op.Data.(map[string]any); ok {
			c.channels[ch.Key()] = data
		}

		for _, id := range ch.MateIDs() {
			if id == c.ID() {
				continue
			}
			iConn := c.server.Connections().Get(id)
			if iConn == nil {
				continue
			}
			if err := iConn.Send(map[string]any{
				"__lxws_channel_event__": "mateUpdated",
				"channel":                ch.Key(),
				"id":                     c.ID(),
				"data":                   c.SharedDataForChannel(ch),
			}, "text", false); err != nil {
				c.server.LifecycleError("mateUpdated send error for %s: %v", id, err)
			}
		}
	}
	//TODO event

	//TODO fmt
	fmt.Printf("OP %v\n", op)
}

func hybi10Encode(payload []byte, opcode byte, masked bool) ([]byte, error) {
	payloadLen := len(payload)

	// First byte: FIN (1) + RSV1-3 (0) + opcode (4 bits)
	first := byte(0x80) | (opcode & 0x0F) // FIN = 1

	// Build the second byte and potencial extended length
	var header []byte
	header = append(header, first)

	switch {
	case payloadLen <= 125:
		b := byte(payloadLen)
		if masked {
			b |= 0x80
		}
		header = append(header, b)
	case payloadLen <= 0xFFFF:
		// 126 + 2 bytes length
		b := byte(126)
		if masked {
			b |= 0x80
		}
		header = append(header, b)
		ext := make([]byte, 2)
		binary.BigEndian.PutUint16(ext, uint16(payloadLen))
		header = append(header, ext...)
	default:
		// 127 + 8 bytes length
		b := byte(127)
		if masked {
			b |= 0x80
		}
		header = append(header, b)
		ext := make([]byte, 8)
		binary.BigEndian.PutUint64(ext, uint64(payloadLen))
		header = append(header, ext...)
	}

	// For mask — gen 4 bytes and mask payload
	if masked {
		maskKey := make([]byte, 4)
		if _, err := rand.Read(maskKey); err != nil {
			return nil, fmt.Errorf("generate mask: %w", err)
		}
		header = append(header, maskKey...)

		// Masked payload
		maskedPayload := make([]byte, payloadLen)
		for i := range payloadLen {
			maskedPayload[i] = payload[i] ^ maskKey[i%4]
		}
		return append(header, maskedPayload...), nil
	}

	// Not masked payload (server -> client)
	return append(header, payload...), nil
}

func hybi10Decode(reader *bufio.Reader) (opcode byte, payload []byte, fin bool, err error) {
	// Read first two bytes
	hdr := make([]byte, 2)
	if _, err = io.ReadFull(reader, hdr); err != nil {
		return 0, nil, false, fmt.Errorf("read header: %w", err)
	}

	first := hdr[0]
	second := hdr[1]

	fin = (first & 0x80) != 0
	opcode = first & 0x0F
	mask := (second & 0x80) != 0
	payloadLen := int(second & 0x7F)

	// Extended payload length
	switch payloadLen {
	case 126:
		ext := make([]byte, 2)
		if _, err = io.ReadFull(reader, ext); err != nil {
			return 0, nil, false, fmt.Errorf("read ext16: %w", err)
		}
		payloadLen = int(binary.BigEndian.Uint16(ext))
	case 127:
		ext := make([]byte, 8)
		if _, err = io.ReadFull(reader, ext); err != nil {
			return 0, nil, false, fmt.Errorf("read ext64: %w", err)
		}
		// Limit: cast to int
		payloadLen64 := binary.BigEndian.Uint64(ext)
		if payloadLen64 > (1<<31 - 1) {
			return 0, nil, false, errors.New("payload too large")
		}
		payloadLen = int(payloadLen64)
	}

	// Mask key (if exists)
	var maskKey []byte
	if mask {
		maskKey = make([]byte, 4)
		if _, err = io.ReadFull(reader, maskKey); err != nil {
			return 0, nil, false, fmt.Errorf("read mask: %w", err)
		}
	}

	// Payload
	payload = make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err = io.ReadFull(reader, payload); err != nil {
			return 0, nil, false, fmt.Errorf("read payload: %w", err)
		}
	}

	// Unmask if needed (client->server frames MUST be masked)
	if mask && payloadLen > 0 {
		for i := 0; i < payloadLen; i++ {
			payload[i] ^= maskKey[i%4]
		}
	}

	return opcode, payload, fin, nil
}

func RandHash() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// fallback (маловероятно), но так хотя бы не будет всех нулей
		return hex.EncodeToString(b)
	}
	return hex.EncodeToString(b)
}
