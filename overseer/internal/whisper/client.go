// SPDX-License-Identifier: AGPL-3.0-only
package whisper

import (
	"bytes"
	"crypto/ecdh"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Client drives a whisper session from the far side of the handshake
type Client struct {
	base    string
	http    *http.Client
	key     SessionKey
	ephPriv *ecdh.PrivateKey
	shardID string
}

// NewClient prepares a client for one Overseer base URL
func NewClient(baseURL string, insecure bool) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" {
		return nil, fmt.Errorf("whisper: client requires an https base URL")
	}
	transport := &http.Transport{}
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // dev only
	}
	return &Client{base: baseURL, http: &http.Client{Transport: transport}}, nil
}

// Handshake runs the full key exchange and registers this client as a Shard
func (c *Client) Handshake(hello Hello) (Ack, error) {
	var ack Ack
	var start StartResp
	if err := c.getJSON("/whisper/v1/handshake/start", &start); err != nil {
		return ack, err
	}
	ephPriv, err := NewEphemeral()
	if err != nil {
		return ack, err
	}
	c.ephPriv = ephPriv
	key, err := DeriveSessionKey(ephPriv, start.EphPub)
	if err != nil {
		return ack, err
	}
	c.key = key
	env, err := Seal(key, mustJSON(hello))
	if err != nil {
		return ack, err
	}
	req := CompleteReq{
		Session: start.Session,
		EphPub:  PubHex(ephPriv.PublicKey()),
		Nonce:   env.Nonce,
		CT:      env.CT,
	}
	var respEnv Envelope
	if err := c.postJSON("/whisper/v1/handshake/complete", req, &respEnv); err != nil {
		return ack, err
	}
	ackPlain, err := Open(key, respEnv)
	if err != nil {
		return ack, err
	}
	if err := json.Unmarshal(ackPlain, &ack); err != nil {
		return ack, err
	}
	c.shardID = ack.ShardID
	return ack, nil
}

// Word sends one sealed task and returns the sealed answer
func (c *Client) Word(word string, args map[string]string) (WordResult, error) {
	var res WordResult
	env, err := Seal(c.key, mustJSON(Word{ShardID: c.shardID, Word: word, Args: args}))
	if err != nil {
		return res, err
	}
	req := wordRequest{ShardID: c.shardID, Envelope: env}
	var respEnv Envelope
	if err := c.postJSON("/whisper/v1/word", req, &respEnv); err != nil {
		return res, err
	}
	resPlain, err := Open(c.key, respEnv)
	if err != nil {
		return res, err
	}
	if err := json.Unmarshal(resPlain, &res); err != nil {
		return res, err
	}
	return res, nil
}

func (c *Client) getJSON(path string, out any) error {
	resp, err := c.http.Get(c.base + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decode(resp, out)
}

func (c *Client) postJSON(path string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	resp, err := c.http.Post(c.base+path, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decode(resp, out)
}

func decode(resp *http.Response, out any) error {
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var msg map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&msg)
		return fmt.Errorf("whisper: server returned %s: %s", resp.Status, msg["error"])
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
