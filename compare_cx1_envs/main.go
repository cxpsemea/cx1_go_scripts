package main

import (
	//"crypto/tls"
	//"flag"

	"crypto/tls"
	"flag"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

func main() {
	os.Exit(mainRunner())
}

func mainRunner() int {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	myformatter := &easy.Formatter{}
	myformatter.TimestampFormat = "2006-01-02 15:04:05.000"
	myformatter.LogFormat = "[%lvl%][%time%] %msg%\n"
	logger.SetFormatter(myformatter)
	logger.SetOutput(os.Stdout)

	logger.Info("Starting")

	LogLevel := flag.String("log", "INFO", "Log level: TRACE, DEBUG, INFO, WARNING, ERROR, FATAL")

	APIKey1 := flag.String("apikey1", "", "CheckmarxOne API Key (if not using client id/secret)")
	ClientID1 := flag.String("client1", "", "CheckmarxOne Client ID (if not using API Key)")
	ClientSecret1 := flag.String("secret1", "", "CheckmarxOne Client Secret (if not using API Key)")
	Cx1URL1 := flag.String("cx1", "", "Optional: CheckmarxOne platform URL")
	IAMURL1 := flag.String("iam1", "", "Optional: CheckmarxOne IAM URL")
	Tenant1 := flag.String("tenant1", "", "Optional: CheckmarxOne tenant")
	Proxy1 := flag.String("proxy1", "", "Optional: Proxy to use when connecting to CheckmarxOne")

	APIKey2 := flag.String("apikey2", "", "CheckmarxOne API Key (if not using client id/secret)")
	ClientID2 := flag.String("client2", "", "CheckmarxOne Client ID (if not using API Key)")
	ClientSecret2 := flag.String("secret2", "", "CheckmarxOne Client Secret (if not using API Key)")
	Cx1URL2 := flag.String("cx2", "", "Optional: CheckmarxOne platform URL")
	IAMURL2 := flag.String("iam2", "", "Optional: CheckmarxOne IAM URL")
	Tenant2 := flag.String("tenant2", "", "Optional: CheckmarxOne tenant")
	Proxy2 := flag.String("proxy2", "", "Optional: Proxy to use when connecting to CheckmarxOne")

	Roles := flag.String("roles", "", "List of comma-separated roles to compare")

	flag.Parse()

	switch strings.ToUpper(*LogLevel) {
	case "TRACE":
		logger.Info("Setting log level to TRACE")
		logger.SetLevel(logrus.TraceLevel)
	case "DEBUG":
		logger.Info("Setting log level to DEBUG")
		logger.SetLevel(logrus.DebugLevel)
	case "INFO":
		logger.Info("Setting log level to INFO")
		logger.SetLevel(logrus.InfoLevel)
	case "WARNING":
		logger.Info("Setting log level to WARNING")
		logger.SetLevel(logrus.WarnLevel)
	case "ERROR":
		logger.Info("Setting log level to ERROR")
		logger.SetLevel(logrus.ErrorLevel)
	case "FATAL":
		logger.Info("Setting log level to FATAL")
		logger.SetLevel(logrus.FatalLevel)
	default:
		logger.Info("Log level set to default: INFO")
	}

	if *Roles == "" {
		logger.Fatalf("Required parameter roles is missing")
	}

	httpClient1 := &http.Client{}
	if *Proxy1 != "" {
		proxyURL, err := url.Parse(*Proxy1)
		if err != nil {
			logger.Fatalf("Failed to parse proxy url %v: %s", *Proxy1, err)
		}
		transport := &http.Transport{}
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		httpClient1.Transport = transport
		logger.Infof("Running with proxy1: %v", *Proxy1)
	}
	httpClient2 := &http.Client{}
	if *Proxy2 != "" {
		proxyURL, err := url.Parse(*Proxy2)
		if err != nil {
			logger.Fatalf("Failed to parse proxy url %v: %s", *Proxy2, err)
		}
		transport := &http.Transport{}
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		httpClient2.Transport = transport
		logger.Infof("Running with proxy2: %v", *Proxy2)
	}

	var cx1client1, cx1client2 *Cx1ClientGo.Cx1Client
	var err error

	if *APIKey1 != "" {
		cx1client1, err = Cx1ClientGo.NewAPIKeyClient(httpClient1, *Cx1URL1, *IAMURL1, *Tenant1, *APIKey1, logger)
	} else {
		cx1client1, err = Cx1ClientGo.NewOAuthClient(httpClient1, *Cx1URL1, *IAMURL1, *Tenant1, *ClientID1, *ClientSecret1, logger)
	}
	if err != nil {
		logger.Fatalf("Failed to create client #1 for %v: %s", *Tenant1, err)
	}
	logger.Infof("Connected client #1 with %v", cx1client1.String())

	if *APIKey2 != "" {
		cx1client2, err = Cx1ClientGo.NewAPIKeyClient(httpClient2, *Cx1URL2, *IAMURL2, *Tenant2, *APIKey2, logger)
	} else {
		cx1client2, err = Cx1ClientGo.NewOAuthClient(httpClient2, *Cx1URL2, *IAMURL2, *Tenant2, *ClientID2, *ClientSecret2, logger)
	}
	if err != nil {
		logger.Fatalf("Failed to create client #2 for %v: %s", *Tenant2, err)
	}
	logger.Infof("Connected client #2 with %v", cx1client2.String())

	rolesToCheck := strings.Split(*Roles, ",")

	diffs := 0

	for _, r := range rolesToCheck {
		role1, err1 := cx1client1.GetRoleByName(r)
		role2, err2 := cx1client2.GetRoleByName(r)

		if err1 == nil && err2 == nil {
			if role1.Composite {
				subroles, err := cx1client1.GetRoleComposites(&role1)
				if err != nil {
					logger.Errorf("Failed to get sub-roles for %v role %v: %s", *Tenant1, role1.String(), err)
				}
				role1.SubRoles = subroles
			}

			if role2.Composite {
				subroles, err := cx1client2.GetRoleComposites(&role2)
				if err != nil {
					logger.Errorf("Failed to get sub-roles for %v role %v: %s", *Tenant2, role2.String(), err)
				}
				role2.SubRoles = subroles
			}

			if !compareRoles(*Tenant1, role1, *Tenant2, role2, logger) {
				diffs++
			}
		} else if err1 != nil && err2 != nil {
			logger.Warnf("Failed to get role %v from both %v and %v", r, *Tenant1, *Tenant2)
		} else {
			if err1 != nil {
				logger.Errorf("Role %v exists in %v but not in %v", r, *Tenant2, *Tenant1)
			} else {
				logger.Errorf("Role %v exists in %v but not in %v", r, *Tenant1, *Tenant2)
			}
			diffs++
		}
	}

	return diffs
}

func compareRoles(tenant1 string, role1 Cx1ClientGo.Role, tenant2 string, role2 Cx1ClientGo.Role, logger *logrus.Logger) bool {
	if !role1.Composite && !role2.Composite {
		logger.Infof("Role %v exists in both tenants and does not contain sub-roles", role1.Name)
		return true
	}

	var missing, common, extra []string
	same := true
	var r1, r2 []string
	for _, r := range role1.SubRoles {
		r1 = append(r1, r.Name)
	}
	for _, r := range role2.SubRoles {
		r2 = append(r2, r.Name)
	}

	for _, r := range r1 {
		if !slices.Contains(r2, r) { // sub-role is in tenant1 but not tenant2
			extra = append(extra, r)
			//logger.Errorf("Role %v in %v contains sub-role %v while role %v in %v does not", role1.String(), tenant1, r, role2.String(), tenant2)
			same = false
		} else {
			common = append(common, r)
		}
	}

	for _, r := range r2 {
		if !slices.Contains(r1, r) { // sub-role is in tenant2 but not tenant1
			missing = append(missing, r)
			//logger.Errorf("Role %v in %v contains sub-role %v while role %v in %v does not", role2.String(), tenant2, r, role1.String(), tenant1)
			same = false
		}
	}

	if same {
		logger.Infof("Role %v is the same between %v and %v", role1.Name, tenant1, tenant2)
	} else {
		logger.Warnf("Role %v is different between %v and %v", role1.Name, tenant1, tenant2)
	}

	if len(common) > 0 {
		logger.Infof(" - %d sub-roles in common: %v", len(common), strings.Join(common, ", "))
	}

	if len(missing) > 0 {
		logger.Warnf(" - %d sub-roles are missing from %v: %v", len(missing), tenant1, strings.Join(missing, ", "))
	}

	if len(extra) > 0 {
		logger.Warnf(" - %d sub-roles are extra in %v: %v", len(extra), tenant1, strings.Join(extra, ", "))
	}

	return same
}
