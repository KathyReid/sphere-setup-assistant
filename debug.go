// +build !release

package main

import "github.com/bugsnag/bugsnag-go"

func init() {
	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       "1e2278d66bee6cbd579606e2a0e623f3",
		ReleaseStage: "development",
	})
}
