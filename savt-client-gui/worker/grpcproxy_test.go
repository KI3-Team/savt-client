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

package worker_test

import (
	"savt-client/savt-client-api/savt"
	"savt-client/savt-client-gui/worker"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGRPCProxy_SetAndGetConfig(t *testing.T) {
	serverAddr := "k02:40001"
	proxy, err := worker.NewGRPCProxy(serverAddr)
	if err != nil {
		t.Fatalf("failed to create GRPCProxy: %v", err)
	}
	defer proxy.Close()

	config, err := proxy.GetConfig()
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	assert.NoError(t, err, "GetConfig should not return an error")
	assert.NotNil(t, config, "GetConfig response should not be nil")
	setConfigEntry := &savt.Config{
		CompleteInterval:        config.CompleteInterval,
		IncompleteRetryInterval: config.IncompleteRetryInterval,
		CheckNetworkInterval:    config.CheckNetworkInterval,
		WaitAfterChangeInterval: config.WaitAfterChangeInterval,
		RetryLimit:              config.RetryLimit,
		IsPublic:                config.IsPublic,
	}
	_, err = proxy.SetConfig(setConfigEntry)
	assert.NoError(t, err, "SetConfig should not return an error")

	newConfig, err := proxy.GetConfig()
	if err != nil {
		t.Fatalf("Failed to get new config: %v", err)
	}
	assert.Equal(t, setConfigEntry.CompleteInterval, newConfig.CompleteInterval, "CompleteInterval should match")
	assert.Equal(t, setConfigEntry.IncompleteRetryInterval, newConfig.IncompleteRetryInterval, "IncompleteRetryInterval should match")
	assert.Equal(t, setConfigEntry.CheckNetworkInterval, newConfig.CheckNetworkInterval, "CheckNetworkInterval should match")
	assert.Equal(t, setConfigEntry.WaitAfterChangeInterval, newConfig.WaitAfterChangeInterval, "WaitAfterChangeInterval should match")
	assert.Equal(t, setConfigEntry.RetryLimit, newConfig.RetryLimit, "RetryLimit should match")
	assert.Equal(t, setConfigEntry.IsPublic, newConfig.IsPublic, "IsPublic should match")
}
