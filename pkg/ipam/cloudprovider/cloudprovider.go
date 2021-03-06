/*
 * Tencent is pleased to support the open source community by making TKEStack available.
 *
 * Copyright (C) 2012-2019 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */
package cloudprovider

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	glog "k8s.io/klog"
	"tkestack.io/galaxy/pkg/ipam/cloudprovider/rpc"
)

var kacp = keepalive.ClientParameters{
	Time:                2 * time.Minute, // send pings every 2 minutes if there is no activity
	Timeout:             time.Minute,     // wait 1 minute for ping ack before considering the connection dead
	PermitWithoutStream: true,            // send pings even without active streams
}

// CloudProvider is a floatingip vendor, such as public cloud eni provider
type CloudProvider interface {
	AssignIP(in *rpc.AssignIPRequest) (*rpc.AssignIPReply, error)
	UnAssignIP(in *rpc.UnAssignIPRequest) (*rpc.UnAssignIPReply, error)
}

type grpcCloudProvider struct {
	init              sync.Once
	cloudProviderAddr string
	client            rpc.IPProviderServiceClient
	timeout           time.Duration
}

// NewGRPCCloudProvider creates a grpcCloudProvider
func NewGRPCCloudProvider(cloudProviderAddr string) CloudProvider {
	return &grpcCloudProvider{
		timeout:           time.Second * 60,
		cloudProviderAddr: cloudProviderAddr,
	}
}

func (p *grpcCloudProvider) connect() {
	p.init.Do(func() {
		glog.V(3).Infof("dial cloud provider with address %s", p.cloudProviderAddr)
		conn, err := grpc.Dial(p.cloudProviderAddr, grpc.WithDialer(
			func(addr string, timeout time.Duration) (net.Conn, error) {
				return net.DialTimeout("tcp", addr, timeout)
			}), grpc.WithInsecure(), grpc.WithKeepaliveParams(kacp))
		if err != nil {
			glog.Fatalf("failed to connect to cloud provider %s: %v", p.cloudProviderAddr, err)
		}
		p.client = rpc.NewIPProviderServiceClient(conn)
	})
}

func (p *grpcCloudProvider) AssignIP(in *rpc.AssignIPRequest) (reply *rpc.AssignIPReply, err error) {
	p.connect()
	glog.V(5).Infof("AssignIP %v", in)

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	reply, err = p.client.AssignIP(ctx, in)
	glog.V(5).Infof("request %v, reply %v, err %v", in, reply, err)
	if err != nil || reply == nil || !reply.Success {
		err = fmt.Errorf("AssignIP for %v failed: reply %v, err %v", in, reply, err)
		glog.V(5).Info(err)
	}
	return
}

func (p *grpcCloudProvider) UnAssignIP(in *rpc.UnAssignIPRequest) (reply *rpc.UnAssignIPReply, err error) {
	p.connect()
	glog.V(5).Infof("UnAssignIP %v", in)

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	reply, err = p.client.UnAssignIP(ctx, in)
	glog.V(5).Infof("request %v, reply %v, err %v", in, reply, err)
	if err != nil || reply == nil || !reply.Success {
		err = fmt.Errorf("UnAssignIP for %v failed: reply %v, err %v", in, reply, err)
		glog.V(5).Info(err)
	}
	return
}
