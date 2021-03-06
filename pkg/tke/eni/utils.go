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
package eni

import (
	"strconv"
	"strings"
	"syscall"

	"github.com/dbdd4us/qcloudapi-sdk-go/metadata"
	log "k8s.io/klog"
)

func getENIIndex(ifName string) (int, error) {
	numStr := strings.TrimLeft(ifName, devPrefix)
	num, err := strconv.ParseInt(numStr, 10, 32)
	if err != nil {
		return 0, err
	}

	return int(num), nil
}

func getENIMetaMap(metaCli *metadata.MetaData) (eniMetaMap map[string]*eniMeta, err error) {
	var macList []string
	var primaryMac string

	eniMetaMap = make(map[string]*eniMeta)
	macList, err = metaCli.EniMacs()
	if err != nil {
		return
	}

	primaryMac, err = metaCli.Mac()
	primaryMac = strings.ToLower(primaryMac)
	if err != nil {
		return
	}
	for _, mac := range macList {
		imac := strings.ToLower(mac)
		enim := &eniMeta{}

		if imac == primaryMac {
			enim.Primary = true
		} else {
			enim.Primary = false
		}
		enim.Mac = imac

		enim.PrimaryIp, err = metaCli.EniPrimaryIpv4(mac)
		if err != nil {
			return
		}
		enim.Mask, err = metaCli.EniIpv4SubnetMask(mac, enim.PrimaryIp)
		if err != nil {
			return
		}
		enim.GateWay, err = metaCli.EniIpv4GateWay(mac, enim.PrimaryIp)
		if err != nil {
			return
		}
		enim.LocalIpList, err = metaCli.EniIpv4List(mac)
		if err != nil {
			return
		}
		log.Infof("Get eni metadata: %+v", enim)
		eniMetaMap[imac] = enim
	}
	return
}

// isNotExistsError returns true if the error type is syscall.ESRCH
// This helps us determine if we should ignore this error as the route
// that we want to cleanup has been deleted already routing table
func IsNotExistsError(err error) bool {
	if errno, ok := err.(syscall.Errno); ok {
		return errno == syscall.ESRCH
	}
	return false
}

// IsFileExistsError returns true if the error type is syscall.EEXIST
// This helps us determine if we should ignore this error as the route
// we want to add has been added already in routing table
func IsFileExistsError(err error) bool {
	if errno, ok := err.(syscall.Errno); ok {
		return errno == syscall.EEXIST
	}
	return false
}
