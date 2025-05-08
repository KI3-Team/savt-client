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

package worker

import (
	"context"
	"fmt"
	"log"
	savt "savt-client/savt-client-api/savt"
	"savt-client/savt-client-gui/errors"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GRPCProxy struct {
	client         savt.SavtIpcClient
	conn           *grpc.ClientConn
	onError        func(*errors.ErrorMessage)
	defaultTimeout time.Duration
}

// NewGRPCProxy creates and initializes a gRPC proxy object
func NewGRPCProxy(serverAddr string, onError ...func(*errors.ErrorMessage)) (*GRPCProxy, error) {
	if len(onError) == 0 {
		onError = append(onError, defaultOnError)
	}
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("could not connect to gRPC server: %v", err)
	}
	client := savt.NewSavtIpcClient(conn)
	return &GRPCProxy{
		client:         client,
		conn:           conn,
		onError:        onError[0],
		defaultTimeout: 10 * time.Second, // default timeout
	}, nil
}

// default error callback
func defaultOnError(errMsg *errors.ErrorMessage) {
	log.Printf("Error occurred: %s", errMsg.Error())
}

// Close closes the gRPC connection
func (proxy *GRPCProxy) Close() {
	if err := proxy.conn.Close(); err != nil {
		log.Printf("failed to close gRPC connection: %v", err)
	}
}

// CallWithContext wraps gRPC method calls and error handling
func (proxy *GRPCProxy) CallWithContext(ctx context.Context, methodName string, request interface{}, callFunc func(ctx context.Context, request interface{}) (interface{}, error)) (interface{}, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), proxy.defaultTimeout)
		defer cancel()
	}
	startTime := time.Now()
	response, err := callFunc(ctx, request)
	duration := time.Since(startTime)
	if err != nil {
		errMsg := errors.NewErrorMessage(errors.ConnectionError, fmt.Sprintf("gRPC call to %s failed: %v", methodName, err))
		if proxy.onError != nil {
			proxy.onError(errMsg)
		}
		log.Printf("Error occurred during gRPC call to %s. Duration: %v", methodName, duration)
		return nil, errMsg
	}
	return response, nil
}

// Home

func (proxy *GRPCProxy) Echo(echoRequest *savt.EchoRequest) (*savt.EchoRequest, error) {
	response, err := proxy.CallWithContext(context.TODO(), "Echo", echoRequest, func(ctx context.Context, req interface{}) (interface{}, error) {
		return proxy.client.Echo(ctx, echoRequest)
	})
	if err != nil {
		return nil, err
	}
	return response.(*savt.EchoRequest), nil
}

func (proxy *GRPCProxy) GetStatus(jobIdRequest *savt.JobIdRequest) (*savt.Job, error) {
	response, err := proxy.CallWithContext(context.TODO(), "GetStatus", jobIdRequest, func(ctx context.Context, req interface{}) (interface{}, error) {
		return proxy.client.GetStatus(ctx, jobIdRequest)
	})
	if err != nil {
		return nil, err
	}
	return response.(*savt.Job), nil
}

func (proxy *GRPCProxy) Start() (*savt.Job, error) {
	response, err := proxy.CallWithContext(context.TODO(), "Start", nil, func(ctx context.Context, req interface{}) (interface{}, error) {
		return proxy.client.Start(ctx, nil)
	})
	if err != nil {
		return nil, err
	}
	return response.(*savt.Job), nil
}
func (proxy *GRPCProxy) Kill(jobIdRequest *savt.JobIdRequest) (*savt.Job, error) {
	response, err := proxy.CallWithContext(context.TODO(), "Kill", jobIdRequest, func(ctx context.Context, req interface{}) (interface{}, error) {
		return proxy.client.Kill(ctx, jobIdRequest)
	})
	if err != nil {
		return nil, err
	}
	return response.(*savt.Job), nil
}

func (proxy *GRPCProxy) ReadLog(ctx context.Context, jobIdRequest *savt.JobIdRequest) (grpc.ServerStreamingClient[savt.LogEntry], error) {
	response, err := proxy.CallWithContext(ctx, "ReadLog", jobIdRequest, func(ctx context.Context, req interface{}) (interface{}, error) {
		return proxy.client.ReadLog(ctx, jobIdRequest)
	})
	if err != nil {
		return nil, err
	}
	return response.(grpc.ServerStreamingClient[savt.LogEntry]), nil
}

// History
func (proxy *GRPCProxy) GetHistory() (*savt.GetHistoryResponse, error) {
	response, err := proxy.CallWithContext(context.TODO(), "GetHistory", nil, func(ctx context.Context, req interface{}) (interface{}, error) {
		return proxy.client.GetHistory(ctx, nil)
	})
	if err != nil {
		return nil, err
	}
	return response.(*savt.GetHistoryResponse), nil
}

// Settings
func (proxy *GRPCProxy) SetConfig(request *savt.Config) (*emptypb.Empty, error) {
	_, err := proxy.CallWithContext(context.TODO(), "SetConfig", request, func(ctx context.Context, req interface{}) (interface{}, error) {
		return proxy.client.SetConfig(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (proxy *GRPCProxy) GetConfig() (*savt.Config, error) {
	response, err := proxy.CallWithContext(context.TODO(), "GetConfig", nil, func(ctx context.Context, req interface{}) (interface{}, error) {
		return proxy.client.GetConfig(ctx, nil)
	})
	if err != nil {
		return nil, err
	}
	return response.(*savt.Config), nil
}
