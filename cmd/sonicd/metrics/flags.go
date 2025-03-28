package metrics

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"
	"github.com/ethereum/go-ethereum/metrics/influxdb"
	"gopkg.in/urfave/cli.v1"
)

var (
	MetricsEnabledFlag = cli.BoolFlag{
		Name:  "metrics",
		Usage: "Enable metrics collection and reporting",
	}
	MetricsEnabledExpensiveFlag = cli.BoolFlag{
		Name:  "metrics.expensive",
		Usage: "Enable expensive metrics collection and reporting",
	}
	// MetricsHTTPFlag defines the endpoint for a stand-alone metrics HTTP endpoint.
	// Since the pprof service enables sensitive/vulnerable behavior, this allows a user
	// to enable a public-OK metrics endpoint without having to worry about ALSO exposing
	// other profiling behavior or information.
	MetricsHTTPFlag = cli.StringFlag{
		Name:  "metrics.addr",
		Usage: "Enable stand-alone metrics HTTP server listening interface",
		Value: metrics.DefaultConfig.HTTP,
	}
	MetricsPortFlag = cli.IntFlag{
		Name:  "metrics.port",
		Usage: "Metrics HTTP server listening port",
		Value: metrics.DefaultConfig.Port,
	}
	MetricsEnableInfluxDBFlag = cli.BoolFlag{
		Name:  "metrics.influxdb",
		Usage: "Enable metrics export/push to an external InfluxDB database",
	}
	MetricsInfluxDBEndpointFlag = cli.StringFlag{
		Name:  "metrics.influxdb.endpoint",
		Usage: "InfluxDB API endpoint to report metrics to",
		Value: metrics.DefaultConfig.InfluxDBEndpoint,
	}
	MetricsInfluxDBDatabaseFlag = cli.StringFlag{
		Name:  "metrics.influxdb.database",
		Usage: "InfluxDB database name to push reported metrics to",
		Value: metrics.DefaultConfig.InfluxDBDatabase,
	}
	MetricsInfluxDBUsernameFlag = cli.StringFlag{
		Name:  "metrics.influxdb.username",
		Usage: "Username to authorize access to the database",
		Value: metrics.DefaultConfig.InfluxDBUsername,
	}
	MetricsInfluxDBPasswordFlag = cli.StringFlag{
		Name:  "metrics.influxdb.password",
		Usage: "Password to authorize access to the database",
		Value: metrics.DefaultConfig.InfluxDBPassword,
	}
	// Tags are part of every measurement sent to InfluxDB. Queries on tags are faster in InfluxDB.
	// For example `host` tag could be used so that we can group all nodes and average a measurement
	// across all of them, but also so that we can select a specific node and inspect its measurements.
	// https://docs.influxdata.com/influxdb/v1.4/concepts/key_concepts/#tag-key
	MetricsInfluxDBTagsFlag = cli.StringFlag{
		Name:  "metrics.influxdb.tags",
		Usage: "Comma-separated InfluxDB tags (key/values) attached to all measurements",
		Value: metrics.DefaultConfig.InfluxDBTags,
	}
	MetricsEnableInfluxDBV2Flag = cli.BoolFlag{
		Name:  "metrics.influxdbv2",
		Usage: "Enable metrics export/push to an external InfluxDB v2 database",
	}
	MetricsInfluxDBTokenFlag = cli.StringFlag{
		Name:  "metrics.influxdb.token",
		Usage: "Token to authorize access to the database (v2 only)",
		Value: metrics.DefaultConfig.InfluxDBToken,
	}
	MetricsInfluxDBBucketFlag = cli.StringFlag{
		Name:  "metrics.influxdb.bucket",
		Usage: "InfluxDB bucket name to push reported metrics to (v2 only)",
		Value: metrics.DefaultConfig.InfluxDBBucket,
	}
	MetricsInfluxDBOrganizationFlag = cli.StringFlag{
		Name:  "metrics.influxdb.organization",
		Usage: "InfluxDB organization name (v2 only)",
		Value: metrics.DefaultConfig.InfluxDBOrganization,
	}
)

func SetupMetrics(ctx *cli.Context) error {
	if metrics.Enabled() {
		log.Info("Enabling metrics collection")

		var (
			enableExport   = ctx.GlobalBool(MetricsEnableInfluxDBFlag.Name)
			enableExportV2 = ctx.GlobalBool(MetricsEnableInfluxDBV2Flag.Name)
		)

		if enableExport || enableExportV2 {
			if ctx.GlobalIsSet(MetricsEnableInfluxDBFlag.Name) && ctx.GlobalIsSet(MetricsEnableInfluxDBV2Flag.Name) {
				return fmt.Errorf("flags --%s and %s can't be used at the same time", MetricsEnableInfluxDBFlag.Name, MetricsEnableInfluxDBV2Flag.Name)
			}

			v1FlagIsSet := ctx.GlobalIsSet(MetricsInfluxDBUsernameFlag.Name) ||
				ctx.GlobalIsSet(MetricsInfluxDBPasswordFlag.Name)

			v2FlagIsSet := ctx.GlobalIsSet(MetricsInfluxDBTokenFlag.Name) ||
				ctx.GlobalIsSet(MetricsInfluxDBOrganizationFlag.Name) ||
				ctx.GlobalIsSet(MetricsInfluxDBBucketFlag.Name)

			if enableExport && v2FlagIsSet {
				return fmt.Errorf("flags --influxdb.metrics.organization, --influxdb.metrics.token, --influxdb.metrics.bucket are only available for influxdb-v2")
			} else if enableExportV2 && v1FlagIsSet {
				return fmt.Errorf("flags --influxdb.metrics.username, --influxdb.metrics.password are only available for influxdb-v1")
			}
		}

		var (
			endpoint = ctx.GlobalString(MetricsInfluxDBEndpointFlag.Name)
			database = ctx.GlobalString(MetricsInfluxDBDatabaseFlag.Name)
			username = ctx.GlobalString(MetricsInfluxDBUsernameFlag.Name)
			password = ctx.GlobalString(MetricsInfluxDBPasswordFlag.Name)

			token        = ctx.GlobalString(MetricsInfluxDBTokenFlag.Name)
			bucket       = ctx.GlobalString(MetricsInfluxDBBucketFlag.Name)
			organization = ctx.GlobalString(MetricsInfluxDBOrganizationFlag.Name)
		)

		if enableExport {
			tagsMap := SplitTagsFlag(ctx.GlobalString(MetricsInfluxDBTagsFlag.Name))

			log.Info("Enabling metrics export to InfluxDB")

			go influxdb.InfluxDBWithTags(metrics.DefaultRegistry, 10*time.Second, endpoint, database, username, password, "geth.", tagsMap)
		} else if enableExportV2 {
			tagsMap := SplitTagsFlag(ctx.GlobalString(MetricsInfluxDBTagsFlag.Name))

			log.Info("Enabling metrics export to InfluxDB (v2)")

			go influxdb.InfluxDBV2WithTags(metrics.DefaultRegistry, 10*time.Second, endpoint, token, bucket, organization, "geth.", tagsMap)
		}

		if ctx.GlobalIsSet(MetricsHTTPFlag.Name) {
			address := fmt.Sprintf("%s:%d", ctx.GlobalString(MetricsHTTPFlag.Name), ctx.GlobalInt(MetricsPortFlag.Name))
			log.Info("Enabling stand-alone metrics HTTP endpoint", "address", address)
			exp.Setup(address)
		}
	}
	return nil
}

func SplitTagsFlag(tagsFlag string) map[string]string {
	tags := strings.Split(tagsFlag, ",")
	tagsMap := map[string]string{}

	for _, t := range tags {
		if t != "" {
			kv := strings.Split(t, "=")

			if len(kv) == 2 {
				tagsMap[kv[0]] = kv[1]
			}
		}
	}

	return tagsMap
}
