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
	"strings"

	"github.com/OSC/k8-ldap-configmap/internal/config"
	"github.com/OSC/k8-ldap-configmap/internal/metrics"
	"github.com/OSC/k8-ldap-configmap/internal/utils"
	"github.com/go-kit/kit/log"
	ldap "github.com/go-ldap/ldap/v3"
)

var (
	mapperFactories    = make(map[string]func(config *config.Config, logger log.Logger) Mapper)
	requiredUserAttrs  = make(map[string][]string)
	requiredGroupAttrs = make(map[string][]string)
)

type Mapper interface {
	Name() string
	ConfigMapName() string
	GetData(users *ldap.SearchResult, groups *ldap.SearchResult) (map[string]string, error)
}

func registerMapper(name string, requiredUser []string, requiredGroup []string, factory func(config *config.Config, logger log.Logger) Mapper) {
	mapperFactories[name] = factory
	requiredUserAttrs[name] = requiredUser
	requiredGroupAttrs[name] = requiredGroup
}

func GetMappers(config *config.Config, logger log.Logger) []Mapper {
	mappers := []Mapper{}
	for name, factory := range mapperFactories {
		if utils.SliceContains(config.EnabledMappers, name) {
			mapper := factory(config, log.With(logger, "mapper", name))
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
