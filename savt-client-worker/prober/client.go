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

package prober

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	Client     *resty.Client
	Exception  *APIException
	Addr       string
	Base       string
	Token      string
	Validation *Validation
}

func NewClient(addr string, token string) *Client {
	base := "http://" + addr
	client := resty.New().SetTimeout(10 * time.Second)
	// client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	return &Client{
		Client:     client,
		Exception:  &APIException{},
		Addr:       addr,
		Base:       base,
		Token:      token,
		Validation: &Validation{},
	}
}

func (c *Client) PostValidation() error {
	resp, err := c.Client.R().
		SetResult(c.Validation).
		SetError(c.Exception).
		SetQueryParam("token", c.Token).
		SetQueryParam("version", Version).
		Post(c.Base + "/api/validation")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf(c.Exception.Message)
	}
	return nil
}

func (c *Client) DeleteValidationByID() error {
	resp, err := c.Client.R().
		SetResult(c.Validation).
		SetError(c.Exception).
		SetQueryParam("token", hex.EncodeToString(c.Validation.Token)).
		SetPathParam("id", strconv.FormatInt(c.Validation.ID, 10)).
		SetBody(c.Validation).
		Delete(c.Base + "/api/validation/{id}")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf(c.Exception.Message)
	}
	return nil
}

func (c *Client) GetScheduleByID(spoofIP string) error {
	resp, err := c.Client.R().
		SetResult(c.Validation).
		SetError(c.Exception).
		SetQueryParam("token", hex.EncodeToString(c.Validation.Token)).
		SetQueryParam("isNAT", c.Validation.IsNAT).
		SetQueryParam("spoofIP", spoofIP).
		SetPathParam("id", strconv.FormatInt(c.Validation.ID, 10)).
		SetBody(map[string]string{"time": time.Now().Format(time.RFC3339)}).
		Post(c.Base + "/api/validation/{id}/schedule")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf(c.Exception.Message)
	}
	return nil
}

func (c *Client) StartInboundValidationByID() error {
	resp, err := c.Client.R().
		SetResult(c.Validation).
		SetError(c.Exception).
		SetQueryParam("token", hex.EncodeToString(c.Validation.Token)).
		SetPathParam("id", strconv.FormatInt(c.Validation.ID, 10)).
		Post(c.Base + "/api/validation/{id}/inbound")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf(c.Exception.Message)
	}
	return nil
}
