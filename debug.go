// +build !release

package main

import (
	"github.com/bugsnag/bugsnag-go"
	"github.com/juju/loggo"
	nlog "github.com/ninjasphere/go-ninja/logger"
)

func init() {
	nlog.GetLogger("").SetLogLevel(loggo.DEBUG)

	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       "1e2278d66bee6cbd579606e2a0e623f3",
		ReleaseStage: "development",
	})
}
