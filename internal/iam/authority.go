package iam

import (
	"fmt"
	"net/url"

	"golang.org/x/net/websocket"
)

// dialAuthority creates a new websocket connection to the authority endpoint.
func (c *Client) dialAuthority() (*websocket.Conn, error) {
	u := url.URL{Scheme: "ws", Host: c.config.IAMHost, Path: c.config.AuthorityPath}
	q := u.Query()
	if c.config.AuthorityPassword != "" {
		q.Set("auth", c.config.AuthorityPassword)
	}
	u.RawQuery = q.Encode()

	// The Origin header is required by x/net/websocket
	origin := "http://localhost/"
	return websocket.Dial(u.String(), "", origin)
}

// sendCommand connects to the authority and sends a specific action payload.
func (c *Client) sendCommand(action string, payload interface{}) error {
	ws, err := c.dialAuthority()
	if err != nil {
		return fmt.Errorf("failed to connect to authority: %w", err)
	}
	defer ws.Close()

	req := map[string]interface{}{
		"action":  action,
		"payload": payload,
	}

	if err := websocket.JSON.Send(ws, req); err != nil {
		return fmt.Errorf("failed to send %s request: %w", action, err)
	}

	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := websocket.JSON.Receive(ws, &resp); err != nil {
		return fmt.Errorf("failed to receive %s response: %w", action, err)
	}

	if !resp.Success {
		return fmt.Errorf("authority rejected %s request: %s", action, resp.Message)
	}

	return nil
}

// BanUser instantly bans a user via the IAM Authority WebSocket.
func (c *Client) BanUser(uid string, reason string) error {
	return c.sendCommand("ban", map[string]string{
		"user_id": uid,
		"reason":  reason,
	})
}

// UnbanUser restores access to a banned user via the IAM Authority WebSocket.
func (c *Client) UnbanUser(uid string) error {
	return c.sendCommand("unban", map[string]string{
		"user_id": uid,
	})
}

// UpdateRoles modifies all roles assigned to a user (replaces the array).
func (c *Client) UpdateRoles(uid string, roles []string) error {
	return c.sendCommand("role", map[string]interface{}{
		"user_id": uid,
		"roles":   roles,
	})
}

// AddRole appends a single role to the user's existing roles.
func (c *Client) AddRole(uid string, role string) error {
	return c.sendCommand("role_add", map[string]string{
		"user_id": uid,
		"role":    role,
	})
}

// RemoveRole removes a specific role from the user's roles.
func (c *Client) RemoveRole(uid string, role string) error {
	return c.sendCommand("role_delete", map[string]string{
		"user_id": uid,
		"role":    role,
	})
}
