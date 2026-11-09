// Copyright 2025 KI3
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"savt-client/savt-client-upgrader/internal/device"
	"savt-client/savt-client-upgrader/internal/health"
	"savt-client/savt-client-upgrader/internal/protocol"
	"savt-client/savt-client-upgrader/internal/svcjson"
	"savt-client/savt-client-upgrader/internal/update"
)

// Client maintains the upgrader websocket session.
type Client struct {
	URL               string
	Device            device.Info
	AppVersion        string
	UpgraderVersion   string
	HeartbeatInterval time.Duration
	ReconnectDelay    time.Duration
	Manager           *update.Manager

	mu        sync.Mutex
	conn      *websocket.Conn
	connected bool
	stopCh    chan struct{}
}

// Run connects and serves forever until stop is closed.
func (c *Client) Run(stop <-chan struct{}) {
	c.stopCh = make(chan struct{})
	go func() {
		<-stop
		close(c.stopCh)
		c.closeConn()
	}()

	for {
		select {
		case <-c.stopCh:
			return
		default:
		}
		if err := c.session(); err != nil {
			log.Printf("websocket session ended: %v", err)
		}
		select {
		case <-c.stopCh:
			return
		case <-time.After(c.ReconnectDelay):
		}
	}
}

func (c *Client) session() error {
	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 15 * time.Second,
	}
	conn, _, err := dialer.Dial(c.URL, nil)
	if err != nil {
		return fmt.Errorf("dial %s: %w", c.URL, err)
	}
	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()
	defer c.closeConn()

	log.Printf("websocket connected: %s", c.URL)

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.readLoop()
	}()

	ticker := time.NewTicker(c.HeartbeatInterval)
	defer ticker.Stop()

	if err := c.sendHeartbeat(); err != nil {
		return err
	}

	for {
		select {
		case <-c.stopCh:
			return nil
		case err := <-errCh:
			return err
		case <-ticker.C:
			if err := c.sendHeartbeat(); err != nil {
				return err
			}
		}
	}
}

func (c *Client) readLoop() error {
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return err
		}
		if err := c.handleMessage(data); err != nil {
			log.Printf("handle message: %v", err)
		}
	}
}

func (c *Client) handleMessage(data []byte) error {
	var env protocol.Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	switch env.Type {
	case protocol.TypePing:
		return c.Send(protocol.Envelope{Type: protocol.TypePong})
	case protocol.TypePong:
		return nil
	case protocol.TypeUpdate:
		var cmd protocol.UpdateCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return fmt.Errorf("decode update: %w", err)
		}
		if cmd.Version == "" {
			return fmt.Errorf("update missing version")
		}
		log.Printf("received update command version=%s id=%s", cmd.Version, cmd.UpdateID)
		if c.Manager != nil {
			c.Manager.HandleUpdate(cmd)
		}
		return nil
	case protocol.TypeUpdateCancel:
		var cancel protocol.UpdateCancel
		if err := json.Unmarshal(data, &cancel); err != nil {
			return fmt.Errorf("decode update_cancel: %w", err)
		}
		log.Printf("received update cancel id=%s", cancel.UpdateID)
		if c.Manager != nil {
			if !c.Manager.Cancel(cancel.UpdateID) {
				log.Printf("update cancel ignored: no matching in-flight update")
			}
		}
		return nil
	default:
		log.Printf("ignoring unknown message type %q", env.Type)
		return nil
	}
}

func (c *Client) sendHeartbeat() error {
	c.mu.Lock()
	connected := c.connected
	persistedID := c.Device.ID
	c.mu.Unlock()
	hb := protocol.Heartbeat{
		Type:          protocol.TypeHeartbeat,
		ID:            persistedID,
		MAC:           c.Device.MAC,
		IP:            c.Device.IP,
		AppVersion:    c.AppVersion,
		OS:            c.Device.OS,
		OSVersion:     c.Device.OSVersion,
		Arch:          c.Device.Arch,
		ModulesStatus: health.Modules(connected),
		Timestamp:     time.Now().Unix(),
	}
	// Always report version from install-dir service.json.
	if ver := svcjson.ReadVersion(); ver != "" {
		hb.AppVersion = ver
		c.mu.Lock()
		c.AppVersion = ver
		c.mu.Unlock()
	}
	// Refresh IP / MAC periodically; keep the persisted device id.
	if info, err := device.Collect(); err == nil {
		hb.IP = info.IP
		hb.MAC = info.MAC
		hb.ID = info.ID
		c.mu.Lock()
		c.Device = info
		c.mu.Unlock()
	}
	return c.Send(hb)
}

// SendJSON marshals and writes a websocket text frame.
func (c *Client) Send(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("websocket not connected")
	}
	_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.conn.WriteJSON(v)
}

// SendStatus implements update.StatusSender.
func (c *Client) SendStatus(status protocol.UpdateStatus) error {
	if err := c.Send(status); err != nil {
		log.Printf("failed to send update status: %v", err)
		return err
	}
	return nil
}

// SendCrash reports an unexpected failure.
func (c *Client) SendCrash(stage, errMsg, stack string) error {
	return c.Send(protocol.CrashReport{
		Type:    protocol.TypeCrashReport,
		ID:      c.Device.ID,
		Stage:   stage,
		Error:   errMsg,
		Stack:   stack,
		Version: c.AppVersion,
		TS:      time.Now().Unix(),
	})
}

func (c *Client) closeConn() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

// Connected reports whether the websocket is currently up.
func (c *Client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}
