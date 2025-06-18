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

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/OSC/k8-ldap-configmap/internal/config"
	localldap "github.com/OSC/k8-ldap-configmap/internal/ldap"
	"github.com/OSC/k8-ldap-configmap/internal/mapper"
	"github.com/OSC/k8-ldap-configmap/internal/metrics"
	"github.com/OSC/k8-ldap-configmap/internal/utils"
	"github.com/alecthomas/kingpin/v2"
	ldap "github.com/go-ldap/ldap/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/version"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	appName     = "k8-ldap-configmap"
	metricsPath = "/metrics"
)

var (
	ldapURL               = kingpin.Flag("ldap-url", "LDAP URL").Envar("LDAP_URL").Required().String()
	ldapTLS               = kingpin.Flag("ldap-tls", "Enable TLS connection to LDAP server").Envar("LDAP_TLS").Default("false").Bool()
	ldapTLSVerify         = kingpin.Flag("ldap-tls-verify", "Verify TLS certificate with LDAP server").Envar("LDAP_TLS_VERIFY").Default("true").Bool()
	ldapTLSCACert         = kingpin.Flag("ldap-tls-ca-cert", "TLS CA Cert for LDAP server").Envar("LDAP_TLS_CA_CERT").String()
	ldapGroupBaseDN       = kingpin.Flag("ldap-group-base-dn", "LDAP Group Base DN").Envar("LDAP_GROUP_BASE_DN").Required().String()
	ldapUserBaseDN        = kingpin.Flag("ldap-user-base-dn", "LDAP User Base DN").Envar("LDAP_USER_BASE_DN").Required().String()
	ldapBindDN            = kingpin.Flag("ldap-bind-dn", "LDAP Bind DN").Envar("LDAP_BIND_DN").String()
	ldapBindPassword      = kingpin.Flag("ldap-bind-password", "LDAP Bind Password").Envar("LDAP_BIND_PASSWORD").String()
	ldapGroupFilter       = kingpin.Flag("ldap-group-filter", "LDAP group filter").Default("(objectClass=posixGroup)").Envar("LDAP_GROUP_FILTER").String()
	ldapUserFilter        = kingpin.Flag("ldap-user-filter", "LDAP user filter").Default("(objectClass=posixAccount)").Envar("LDAP_USER_FILTER").String()
	ldapPagedSearch       = kingpin.Flag("ldap-paged-search", "Enable LDAP paged searching").Default("false").Envar("LDAP_PAGED_SEARCH").Bool()
	ldapPagedSearchSize   = kingpin.Flag("ldap-paged-search-size", " LDAP paged search size").Default("1000").Envar("LDAP_PAGED_SEARCH_SIZE").Int()
	ldapMemberScheme      = kingpin.Flag("ldap-member-scheme", "Scheme used to define group members, either memberof, member or memberuid").Default("memberof").Envar("LDAP_MEMBER_SCHEME").String()
	ldapUserAttrMap       = kingpin.Flag("ldap-user-attr-map", "Attribute map for users").Default(config.DefaultUserAttrMap).Envar("LDAP_USER_ATTR_MAP").String()
	ldapGroupAttrMap      = kingpin.Flag("ldap-group-attr-map", "Attribute map for groups").Default(config.DefaultGroupAttrMap).Envar("LDAP_GROUP_ATTR_MAP").String()
	mappersArg            = kingpin.Flag("mappers", "Comma separated list of mappers to generate.").Default("user-uid,user-gid").Envar("MAPPERS").String()
	mappersGroupFilter    = kingpin.Flag("mappers-group-filter", "Comma separated mappers filters map for groups").Default("").Envar("MAPPERS_GROUP_FILTER").String()
	mappersUserFilter     = kingpin.Flag("mappers-user-filter", "Comma separated mappers filters map for users").Default("").Envar("MAPPERS_USER_FILTER").String()
	namespace             = kingpin.Flag("namespace", "namespace for ConfigMaps").Envar("NAMESPACE").Required().String()
	userPrefix            = kingpin.Flag("user-prefix", "Prefix to add to user names").Envar("USER_PREFIX").String()
	interval              = kingpin.Flag("interval", "Duration between sync runs").Default("5m").Envar("INTERLVAL").Duration()
	listenAddress         = kingpin.Flag("listen-address", "Address to listen for HTTP requests").Default(":8080").Envar("LISTEN_ADDRESS").String()
	processMetrics        = kingpin.Flag("process-metrics", "Collect metrics about running process such as CPU and memory and Go stats").Default("true").Envar("PROCESS_METRICS").Bool()
	kubeconfig            = kingpin.Flag("kubeconfig", "Path to kubeconfig when running outside Kubernetes cluster").Default("").Envar("KUBECONFIG").String()
	logLevel              = kingpin.Flag("log-level", "Log level, One of: [debug, info, warn, error]").Default("info").Envar("LOG_LEVEL").Enum(promslog.LevelFlagOptions...)
	logFormat             = kingpin.Flag("log-format", "Log format, One of: [logfmt, json]").Default("logfmt").Envar("LOG_FORMAT").Enum(promslog.FormatFlagOptions...)
	validLdapMemberScheme = []string{"memberof", "member", "memberuid"}
)

func main() {
	kingpin.Version(version.Print(appName))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := setupLogging()
	if logger == nil {
		os.Exit(1)
	}

	var err error
	err = validateArgs(logger)
	if err != nil {
		os.Exit(1)
	}

	var config *rest.Config
	if *kubeconfig == "" {
		logger.Info("Loading in cluster kubeconfig", "kubeconfig", *kubeconfig)
		config, err = rest.InClusterConfig()
	} else {
		logger.Info("Loading kubeconfig", "kubeconfig", *kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	}
	if err != nil {
		logger.Error("Error loading kubeconfig", "err", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error("Unable to generate Clientset", "err", err)
		os.Exit(1)
	}

	c := createConfig()
	mappers := mapper.GetMappers(c, logger)

	logger.Info(fmt.Sprintf("Starting %s", appName), "version", version.Info())
	logger.Info("Build context", "build_context", version.BuildContext())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
	             <head><title>` + appName + `</title></head>
	             <body>
	             <h1>` + appName + `</h1>
	             <p><a href='` + metricsPath + `'>Metrics</a></p>
	             </body>
	             </html>`))
	})
	http.Handle(metricsPath, promhttp.HandlerFor(metrics.MetricGathers(*processMetrics), promhttp.HandlerOpts{}))

	go func() {
		if err := http.ListenAndServe(*listenAddress, nil); err != nil {
			logger.Error("Error starting HTTP server", "err", err)
			os.Exit(1)
		}
	}()

	for {
		var errNum float64
		start := time.Now()
		metrics.MetricLastRun.Set(float64(start.Unix()))
		err = run(mappers, c, clientset, logger)
		metrics.MetricDuration.Set(time.Since(start).Seconds())
		if err != nil {
			errNum = 1
		}
		metrics.MetricError.Set(errNum)
		logger.Debug("Sleeping for interval", "interval", fmt.Sprintf("%.0f", (*interval).Seconds()))
		time.Sleep(*interval)
	}
}

func run(mappers []mapper.Mapper, config *config.Config, clientset kubernetes.Interface, logger *slog.Logger) error {
	l, err := localldap.LDAPConnect(config, logger)
	if err != nil {
		return err
	}
	defer l.Close()

	var groupResults, userResults *ldap.SearchResult
	var groupErr, userErr error
	searchWG := &sync.WaitGroup{}
	searchWG.Add(2)
	go func() {
		defer searchWG.Done()
		if len(config.RequiredGroupAttrs) == 0 {
			return
		}
		groupResults, groupErr = localldap.LDAPGroups(l, config.GroupFilter, config, logger)
	}()
	go func() {
		defer searchWG.Done()
		if len(config.RequiredUserAttrs) == 0 {
			return
		}
		userResults, userErr = localldap.LDAPUsers(l, config.UserFilter, config, logger)
	}()
	searchWG.Wait()
	if groupErr != nil {
		return groupErr
	}
	if userErr != nil {
		return userErr
	}

	errs, _ := errgroup.WithContext(context.Background())
	for _, m := range mappers {
		_m := m
		errs.Go(func() error {
			var err error
			var mapperGroupResults, mapperUserResults *ldap.SearchResult
			if filter, ok := config.MappersGroupFilter[_m.Name()]; ok {
				mapperGroupResults, err = localldap.LDAPGroups(l, filter, config, logger)
				if err != nil {
					return err
				}
			} else {
				mapperGroupResults = groupResults
			}
			if filter, ok := config.MappersUserFilter[_m.Name()]; ok {
				mapperUserResults, err = localldap.LDAPUsers(l, filter, config, logger)
				if err != nil {
					return err
				}
			} else {
				mapperUserResults = userResults
			}
			data, err := _m.GetData(mapperUserResults, mapperGroupResults)
			if err != nil {
				metrics.MetricErrorsTotal.WithLabelValues(_m.Name()).Inc()
				return err
			}
			err = configmap(clientset, _m.ConfigMapName(), data, logger)
			if err != nil {
				metrics.MetricErrorsTotal.WithLabelValues(_m.Name()).Inc()
				return err
			}
			return nil
		})
	}
	return errs.Wait()
}

func configmap(clientset kubernetes.Interface, name string, data map[string]string, logger *slog.Logger) error {
	var err error
	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: *namespace,
		},
		Data: data,
	}
	var action string
	if _, err = clientset.CoreV1().ConfigMaps(*namespace).Get(context.TODO(), name, metav1.GetOptions{}); k8errors.IsNotFound(err) {
		action = "create"
		_, err = clientset.CoreV1().ConfigMaps(*namespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
	} else {
		action = "update"
		_, err = clientset.CoreV1().ConfigMaps(*namespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
	}
	if err == nil {
		logger.Info("ConfigMap sync successful", "action", action, "name", name, "namespace", *namespace)
		metrics.MetricConfigMapKeys.WithLabelValues(name).Set(float64(len(data)))
		configMapJSON, err := json.Marshal(configMap)
		if err != nil {
			logger.Error("Unable to marshall configmap to JSON", "name", name, "namespace", *namespace, "err", err)
			return err
		}
		metrics.MetricConfigMapSize.WithLabelValues(name).Set(float64(len(configMapJSON)))
	} else {
		logger.Error("Failed to sync ConfigMap", "action", action, "name", name, "namespace", *namespace, "err", err)
	}
	return err
}

func createConfig() *config.Config {
	userAttrMap := utils.AttrMap(*ldapUserAttrMap)
	groupAttrMap := utils.AttrMap(*ldapGroupAttrMap)
	enabledMappers := strings.Split(*mappersArg, ",")
	requiredUserAttrs := mapper.RequiredAttrs("user", enabledMappers)
	requiredGroupAttrs := mapper.RequiredAttrs("group", enabledMappers)
	mappersUserFilterMap := utils.AttrMap(*mappersUserFilter)
	mappersGroupFilterMap := utils.AttrMap(*mappersGroupFilter)
	return &config.Config{
		LdapURL:            *ldapURL,
		LdapTLS:            *ldapTLS,
		LdapTLSVerify:      *ldapTLSVerify,
		LdapTLSCACert:      *ldapTLSCACert,
		BindDN:             *ldapBindDN,
		BindPassword:       *ldapBindPassword,
		UserBaseDN:         *ldapUserBaseDN,
		GroupBaseDN:        *ldapGroupBaseDN,
		UserFilter:         *ldapUserFilter,
		GroupFilter:        *ldapGroupFilter,
		UserAttrMap:        userAttrMap,
		GroupAttrMap:       groupAttrMap,
		RequiredUserAttrs:  requiredUserAttrs,
		RequiredGroupAttrs: requiredGroupAttrs,
		PagedSearch:        *ldapPagedSearch,
		PagedSearchSize:    *ldapPagedSearchSize,
		MemberScheme:       *ldapMemberScheme,
		UserPrefix:         *userPrefix,
		EnabledMappers:     enabledMappers,
		MappersUserFilter:  mappersUserFilterMap,
		MappersGroupFilter: mappersGroupFilterMap,
	}
}

func validateArgs(logger *slog.Logger) error {
	errs := []string{}
	validMappers := mapper.ValidMappers()
	enabledMappers := strings.Split(*mappersArg, ",")
	userAttrs := mapper.RequiredAttrs("user", enabledMappers)
	groupAttrs := mapper.RequiredAttrs("group", enabledMappers)
	var err error
	if len(groupAttrs) > 0 {
		groupAttrMap := utils.AttrMap(*ldapGroupAttrMap)
		groupAttrMapKeys := utils.MapKeysStrings(groupAttrMap)
		for _, attr := range groupAttrs {
			if !utils.SliceContains(groupAttrMapKeys, attr) {
				errs = append(errs, fmt.Sprintf("ldap-group-attr-map=\"Missing group attribute map key '%s'\"", attr))
			}
		}
	}
	if len(userAttrs) > 0 {
		userAttrMap := utils.AttrMap(*ldapUserAttrMap)
		userAttrMapKeys := utils.MapKeysStrings(userAttrMap)
		for _, attr := range userAttrs {
			if !utils.SliceContains(userAttrMapKeys, attr) {
				errs = append(errs, fmt.Sprintf("ldap-user-attr-map=\"Missing user attribute map key '%s'\"", attr))
			}
		}
	}
	if (*ldapBindDN != "" && *ldapBindPassword == "") || (*ldapBindDN == "" && *ldapBindPassword != "") {
		errs = append(errs, "ldap-bind=\"Must provide both LDAP Bind DN and Bind Password if either is provided\"")
	}
	if !utils.SliceContains(validLdapMemberScheme, *ldapMemberScheme) {
		errs = append(errs, fmt.Sprintf("ldap-member-scheme=\"LDAP member scheme '%s' invalid\"", *ldapMemberScheme))
	}
	for _, mapper := range strings.Split(*mappersArg, ",") {
		if !utils.SliceContains(validMappers, mapper) {
			errs = append(errs, fmt.Sprintf("mappers=\"Defined mapper %s is not valid\"", mapper))
		}
	}
	mappersUserFilterMap := utils.AttrMap(*mappersUserFilter)
	mappersUserFilterMapKeys := utils.MapKeysStrings(mappersUserFilterMap)
	for _, mapper := range mappersUserFilterMapKeys {
		if !utils.SliceContains(validMappers, mapper) {
			errs = append(errs, fmt.Sprintf("mappers-user-filter=\"Defined mapper %s is not valid for mappers filters users\"", mapper))
		}
	}
	mappersGroupFilterMap := utils.AttrMap(*mappersGroupFilter)
	mappersGroupFilterMapKeys := utils.MapKeysStrings(mappersGroupFilterMap)
	for _, mapper := range mappersGroupFilterMapKeys {
		if !utils.SliceContains(validMappers, mapper) {
			errs = append(errs, fmt.Sprintf("mappers-group-filter=\"Defined mapper %s is not valid for mappers filters groups\"", mapper))
		}
	}
	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, ", "))
		logger.Error(err.Error())
	}
	return err
}

func setupLogging() *slog.Logger {
	level := promslog.NewLevel()
	_ = level.Set(*logLevel)
	format := promslog.NewFormat()
	_ = format.Set(*logFormat)
	promslogConfig := &promslog.Config{
		Level:  level,
		Format: format,
	}
	logger := promslog.New(promslogConfig)
	return logger
}
