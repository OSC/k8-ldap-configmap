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

package ldap

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/url"
	"strings"

	"github.com/OSC/k8-ldap-configmap/internal/config"
	"github.com/OSC/k8-ldap-configmap/internal/utils"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	ldap "github.com/go-ldap/ldap/v3"
)

func LDAPConnect(config *config.Config, logger log.Logger) (*ldap.Conn, error) {
	level.Debug(logger).Log("msg", "Connecting to LDAP", "url", config.LdapURL)
	l, err := ldap.DialURL(config.LdapURL)
	if err != nil {
		level.Error(logger).Log("msg", "Error connecting to LDAP URL", "url", config.LdapURL, "err", err)
		return l, err
	}
	if config.LdapTLS {
		err = LDAPTLS(l, config, logger)
		if err != nil {
			return l, err
		}
	}
	if config.BindDN != "" && config.BindPassword != "" {
		level.Debug(logger).Log("msg", "Binding to LDAP", "url", config.LdapURL, "binddn", config.BindDN)
		err = l.Bind(config.BindDN, config.BindPassword)
		if err != nil {
			level.Error(logger).Log("msg", "Error binding to LDAP", "binddn", config.BindDN, "err", err)
			return l, err
		}
	}
	return l, err
}

func LDAPTLS(l *ldap.Conn, config *config.Config, logger log.Logger) error {
	var err error
	u, err := url.Parse(config.LdapURL)
	if err != nil {
		level.Error(logger).Log("msg", "Error parsing LDAP URL", "url", config.LdapURL, "err", err)
		return err
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		level.Error(logger).Log("msg", "Error getting LDAP host name", "host", u.Host, "err", err)
		return err
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: !config.LdapTLSVerify,
		ServerName:         host,
	}
	if config.LdapTLSCACert != "" {
		caCert := []byte{}
		if strings.HasPrefix(config.LdapTLSCACert, "/") && utils.FileExists(config.LdapTLSCACert) {
			caCert, err = ioutil.ReadFile(config.LdapTLSCACert)
			if err != nil {
				level.Error(logger).Log("msg", "Failed to read CA certificate", "ca-certificate", config.LdapTLSCACert, "err", err)
				return err
			}
		} else {
			caCert = []byte(config.LdapTLSCACert)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}
	level.Debug(logger).Log("msg", "Performing Start TLS with LDAP server")
	err = l.StartTLS(tlsConfig)
	if err != nil {
		level.Error(logger).Log("msg", "Error starting TLS for LDAP connection", "err", err)
	}
	return err
}

func LDAPGroups(l *ldap.Conn, config *config.Config, logger log.Logger) (*ldap.SearchResult, error) {
	attrs := []string{}
	for _, a := range config.RequiredGroupAttrs {
		attrs = append(attrs, config.GroupAttrMap[a])
	}
	switch config.MemberScheme {
	case "member":
		attrs = append(attrs, "member")
	case "memberuid":
		attrs = append(attrs, "memberuid")
	}
	level.Debug(logger).Log("msg", "Running group search", "basedn", config.GroupBaseDN, "filter", config.GroupFilter)
	request := ldap.NewSearchRequest(config.GroupBaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		config.GroupFilter, attrs, nil)
	result, err := LDAPSearch(l, request, "group", config, logger)
	return result, err
}

func LDAPUsers(l *ldap.Conn, config *config.Config, logger log.Logger) (*ldap.SearchResult, error) {
	attrs := []string{}
	for _, a := range config.RequiredUserAttrs {
		attrs = append(attrs, config.UserAttrMap[a])
	}
	if config.MemberScheme == "memberof" {
		attrs = append(attrs, "memberof")
	}
	level.Debug(logger).Log("msg", "Running user search", "basedn", config.UserBaseDN, "filter", config.UserFilter)
	request := ldap.NewSearchRequest(config.UserBaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		config.UserFilter, attrs, nil)
	result, err := LDAPSearch(l, request, "user", config, logger)
	return result, err
}

func LDAPSearch(l *ldap.Conn, request *ldap.SearchRequest, queryType string, config *config.Config, logger log.Logger) (*ldap.SearchResult, error) {
	var result *ldap.SearchResult
	var err error
	if config.PagedSearch {
		result, err = l.SearchWithPaging(request, uint32(config.PagedSearchSize))
	} else {
		result, err = l.Search(request)
	}
	if err != nil {
		level.Error(logger).Log("msg", "Error getting results", "type", queryType, "err", err)
	} else {
		level.Debug(logger).Log("msg", "results", "type", queryType, "count", len(result.Entries))
	}
	return result, err
}
