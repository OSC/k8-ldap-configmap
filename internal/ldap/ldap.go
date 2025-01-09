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
	"log/slog"
	"net"
	"net/url"

	"github.com/OSC/k8-ldap-configmap/internal/config"
	ldap "github.com/go-ldap/ldap/v3"
)

func LDAPConnect(config *config.Config, logger *slog.Logger) (*ldap.Conn, error) {
	logger.Debug("Connecting to LDAP", "url", config.LdapURL)
	l, err := ldap.DialURL(config.LdapURL)
	if err != nil {
		logger.Error("Error connecting to LDAP URL", "url", config.LdapURL, "err", err)
		return l, err
	}
	if config.LdapTLS {
		err = LDAPTLS(l, config, logger)
		if err != nil {
			return l, err
		}
	}
	if config.BindDN != "" && config.BindPassword != "" {
		logger.Debug("Binding to LDAP", "url", config.LdapURL, "binddn", config.BindDN)
		err = l.Bind(config.BindDN, config.BindPassword)
		if err != nil {
			logger.Error("Error binding to LDAP", "binddn", config.BindDN, "err", err)
			return l, err
		}
	}
	return l, err
}

func LDAPTLS(l *ldap.Conn, config *config.Config, logger *slog.Logger) error {
	var err error
	u, err := url.Parse(config.LdapURL)
	if err != nil {
		logger.Error("Error parsing LDAP URL", "url", config.LdapURL, "err", err)
		return err
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		logger.Error("Error getting LDAP host name", "host", u.Host, "err", err)
		return err
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: !config.LdapTLSVerify,
		ServerName:         host,
	}
	if config.LdapTLSCACert != "" {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(config.LdapTLSCACert))
		tlsConfig.RootCAs = caCertPool
	}
	logger.Debug("Performing Start TLS with LDAP server")
	err = l.StartTLS(tlsConfig)
	if err != nil {
		logger.Error("Error starting TLS for LDAP connection", "err", err)
	}
	return err
}

func LDAPGroups(l *ldap.Conn, filter string, config *config.Config, logger *slog.Logger) (*ldap.SearchResult, error) {
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
	logger.Debug("Running group search", "basedn", config.GroupBaseDN, "filter", config.GroupFilter)
	request := ldap.NewSearchRequest(config.GroupBaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter, attrs, nil)
	result, err := LDAPSearch(l, request, "group", config, logger)
	return result, err
}

func LDAPUsers(l *ldap.Conn, filter string, config *config.Config, logger *slog.Logger) (*ldap.SearchResult, error) {
	attrs := []string{}
	for _, a := range config.RequiredUserAttrs {
		attrs = append(attrs, config.UserAttrMap[a])
	}
	if config.MemberScheme == "memberof" {
		attrs = append(attrs, "memberof")
	}
	logger.Debug("Running user search", "basedn", config.UserBaseDN, "filter", config.UserFilter)
	request := ldap.NewSearchRequest(config.UserBaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter, attrs, nil)
	result, err := LDAPSearch(l, request, "user", config, logger)
	return result, err
}

func LDAPSearch(l *ldap.Conn, request *ldap.SearchRequest, queryType string, config *config.Config, logger *slog.Logger) (*ldap.SearchResult, error) {
	var result *ldap.SearchResult
	var err error
	if config.PagedSearch {
		result, err = l.SearchWithPaging(request, uint32(config.PagedSearchSize))
	} else {
		result, err = l.Search(request)
	}
	if err != nil {
		logger.Error("Error getting results", "type", queryType, "err", err)
	} else {
		logger.Debug("results", "type", queryType, "count", len(result.Entries))
	}
	return result, err
}
