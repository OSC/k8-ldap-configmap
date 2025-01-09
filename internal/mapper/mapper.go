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
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/OSC/k8-ldap-configmap/internal/config"
	"github.com/OSC/k8-ldap-configmap/internal/metrics"
	"github.com/OSC/k8-ldap-configmap/internal/utils"
	ldap "github.com/go-ldap/ldap/v3"
)

var (
	mapperFactories    = make(map[string]func(config *config.Config, logger *slog.Logger) Mapper)
	requiredUserAttrs  = make(map[string][]string)
	requiredGroupAttrs = make(map[string][]string)
)

type Mapper interface {
	Name() string
	ConfigMapName() string
	GetData(users *ldap.SearchResult, groups *ldap.SearchResult) (map[string]string, error)
}

type Group struct {
	name string
	gid  int
}

func registerMapper(name string, requiredUser []string, requiredGroup []string, factory func(config *config.Config, logger *slog.Logger) Mapper) {
	mapperFactories[name] = factory
	requiredUserAttrs[name] = requiredUser
	requiredGroupAttrs[name] = requiredGroup
}

func GetMappers(config *config.Config, logger *slog.Logger) []Mapper {
	mappers := []Mapper{}
	for name, factory := range mapperFactories {
		if utils.SliceContains(config.EnabledMappers, name) {
			mapper := factory(config, logger.With("mapper", name))
			mappers = append(mappers, mapper)
			metrics.MetricErrorsTotal.WithLabelValues(mapper.Name())
			metrics.MetricConfigMapSize.WithLabelValues(mapper.ConfigMapName())
			metrics.MetricConfigMapKeys.WithLabelValues(mapper.ConfigMapName())
		}
	}
	return mappers
}

func ValidMappers() []string {
	validMappers := []string{}
	for key := range mapperFactories {
		validMappers = append(validMappers, key)
	}
	return validMappers
}

func RequiredAttrs(attrType string, enabledMappers []string) []string {
	var requiredByType map[string][]string
	if attrType == "user" {
		requiredByType = requiredUserAttrs
	} else {
		requiredByType = requiredGroupAttrs
	}
	attrs := []string{}
	for _, mapper := range enabledMappers {
		required := requiredByType[mapper]
		for _, attr := range required {
			if !utils.SliceContains(attrs, attr) {
				attrs = append(attrs, attr)
			}
		}
	}
	return attrs
}

func ParseDN(dn string) string {
	elements := strings.Split(dn, ",")
	nameElement := elements[0]
	name := strings.Split(nameElement, "=")
	if len(name) != 2 {
		return ""
	}
	return name[1]
}

func GetUserGroups(users *ldap.SearchResult, groups *ldap.SearchResult, config *config.Config, logger *slog.Logger) (map[string][]Group, error) {
	userDNs := make(map[string]string)
	groupDNs := make(map[string]string)
	groupToGid := make(map[string]string)
	gidToGroup := make(map[string]string)
	userGroups := make(map[string][]string)
	data := make(map[string][]Group)

	for _, entry := range users.Entries {
		name := entry.GetAttributeValue(config.UserAttrMap["name"])
		userDNs[strings.ToLower(entry.DN)] = name
	}

	for _, entry := range groups.Entries {
		name := entry.GetAttributeValue(config.GroupAttrMap["name"])
		gid := entry.GetAttributeValue(config.GroupAttrMap["gid"])
		groupDNs[strings.ToLower(entry.DN)] = name
		groupToGid[name] = gid
		gidToGroup[gid] = name
		members := []string{}
		if config.MemberScheme == "member" {
			members = GetGroupsMember(entry.GetAttributeValues("member"), userDNs)
		} else if config.MemberScheme == "memberuid" {
			members = entry.GetAttributeValues("memberUid")
		}
		for _, member := range members {
			key := fmt.Sprintf("%s%s", config.UserPrefix, member)
			groups := []string{}
			if g, ok := userGroups[key]; ok {
				groups = g
			}
			groups = append(groups, name)
			userGroups[key] = groups
		}
	}

	for _, entry := range users.Entries {
		name := entry.GetAttributeValue(config.UserAttrMap["name"])
		key := fmt.Sprintf("%s%s", config.UserPrefix, name)
		gid := entry.GetAttributeValue(config.UserAttrMap["gid"])
		var primaryGroup string
		if g, ok := gidToGroup[gid]; ok {
			primaryGroup = g
		}
		var groups []string
		if g, ok := userGroups[key]; ok {
			groups = g
		}
		if config.MemberScheme == "memberof" {
			groups = GetGroupsMemberOf(entry.GetAttributeValues("memberOf"), groupDNs)
		}
		if !utils.SliceContains(groups, primaryGroup) && primaryGroup != "" {
			groups = append([]string{primaryGroup}, groups...)
		}
		userGroups[key] = groups
	}

	for user, groupNames := range userGroups {
		groups := []Group{}
		for _, groupName := range groupNames {
			group := Group{name: groupName}
			if gid, ok := groupToGid[groupName]; ok {
				gidInt, err := strconv.Atoi(gid)
				if err != nil {
					logger.Error("Unable to parse GID to int", "err", err, "group", groupName, "gid", gid)
					return nil, err
				}
				group.gid = gidInt
			}
			groups = append(groups, group)
		}
		data[user] = groups
	}
	return data, nil
}

func GetGroupsMemberOf(memberOf []string, groupDNs map[string]string) []string {
	groups := []string{}
	for _, m := range memberOf {
		if val, ok := groupDNs[strings.ToLower(m)]; ok {
			groups = append(groups, val)
		}
	}
	return groups
}

func GetGroupsMember(members []string, userDNs map[string]string) []string {
	users := []string{}
	for _, m := range members {
		if val, ok := userDNs[strings.ToLower(m)]; ok {
			users = append(users, val)
		}
	}
	return users
}
