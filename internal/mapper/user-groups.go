// Copyright 2020 Ohio Supercomputer Center
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mapper

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/OSC/k8-ldap-configmap/internal/config"
	"github.com/OSC/k8-ldap-configmap/internal/utils"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	ldap "github.com/go-ldap/ldap/v3"
)

func init() {
	registerMapper("user-groups", []string{"name", "gid"}, []string{"name", "gid"}, NewUserGroupsMapper)
}

func NewUserGroupsMapper(config *config.Config, logger log.Logger) Mapper {
	return &UserGroups{
		config: config,
		logger: logger,
	}
}

type UserGroups struct {
	config *config.Config
	logger log.Logger
}

func (m UserGroups) Name() string {
	return "user-groups"
}

func (m UserGroups) ConfigMapName() string {
	return "user-groups-map"
}

func (m UserGroups) GetData(users *ldap.SearchResult, groups *ldap.SearchResult) (map[string]string, error) {
	level.Debug(m.logger).Log("msg", "Mapper running")
	userDNs := make(map[string]string)
	groupDNs := make(map[string]string)
	gidToGroup := make(map[string]string)
	userGroups := make(map[string][]string)
	data := make(map[string]string)

	for _, entry := range users.Entries {
		name := entry.GetAttributeValue(m.config.UserAttrMap["name"])
		userDNs[strings.ToLower(entry.DN)] = name
	}

	for _, entry := range groups.Entries {
		name := entry.GetAttributeValue(m.config.GroupAttrMap["name"])
		gid := entry.GetAttributeValue(m.config.GroupAttrMap["gid"])
		groupDNs[strings.ToLower(entry.DN)] = name
		gidToGroup[gid] = name
		members := []string{}
		if m.config.MemberScheme == "member" {
			members = m.GetGroupsMember(entry.GetAttributeValues("member"), userDNs)
		} else if m.config.MemberScheme == "memberuid" {
			members = entry.GetAttributeValues("memberUid")
		}
		for _, member := range members {
			groups := []string{}
			groups = append(groups, name)
			userGroups[member] = groups
		}
	}

	for _, entry := range users.Entries {
		name := entry.GetAttributeValue(m.config.UserAttrMap["name"])
		key := fmt.Sprintf("%s%s", m.config.UserPrefix, name)
		gid := entry.GetAttributeValue(m.config.UserAttrMap["gid"])
		var primaryGroup string
		if g, ok := gidToGroup[gid]; ok {
			primaryGroup = g
		}
		var groups []string
		if m.config.MemberScheme == "memberof" {
			groups = m.GetGroupsMemberOf(entry.GetAttributeValues("memberOf"), groupDNs)
		} else if g, ok := userGroups[name]; ok {
			groups = g
		}
		if !utils.SliceContains(groups, primaryGroup) && primaryGroup != "" {
			groups = append([]string{primaryGroup}, groups...)
		}
		userGroups[key] = groups
	}

	for user, groups := range userGroups {
		userGroupsJSON, _ := json.Marshal(groups)
		data[user] = string(userGroupsJSON)
	}
	level.Debug(m.logger).Log("msg", "Mapper complete", "user-groups", len(data))
	return data, nil
}

func (m UserGroups) GetGroupsMemberOf(memberOf []string, groupDNs map[string]string) []string {
	groups := []string{}
	for _, m := range memberOf {
		if val, ok := groupDNs[strings.ToLower(m)]; ok {
			groups = append(groups, val)
		}
	}
	return groups
}

func (m UserGroups) GetGroupsMember(members []string, userDNs map[string]string) []string {
	users := []string{}
	for _, m := range members {
		if val, ok := userDNs[strings.ToLower(m)]; ok {
			users = append(users, val)
		}
	}
	return users
}
