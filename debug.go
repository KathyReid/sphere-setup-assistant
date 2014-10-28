// +build !release

package main

import (
	"github.com/bugsnag/bugsnag-go"
	"github.com/juju/loggo"
)

func init() {
	loggo.GetLogger("").SetLogLevel(loggo.DEBUG)

	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       "1e2278d66bee6cbd579606e2a0e623f3",
		ReleaseStage: "development",
	})
}
