package main

import (
	"go.uber.org/zap"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

/*
The Mountain discovery logic is already nice self contained here:
https://stash.us.cray.com/projects/HMS/repos/hms-mountain-discovery/browse

No point in re-doing all that fine work. Thusly this utility expects to have those bits available for running. In a
production setting this is accomplished by the Docker image literally coping those files from that image. In a
development setting I would recommend checking out that repo and referencing that Python file.
*/

var mountainLoggingRegex = regexp.MustCompile(`.+-([A-Z]+)-(.+)`)

func doMountainDiscovery() {
	command := exec.Command("python3", *mountainDiscoveryScript)
	command.Env = append(os.Environ(),
		"HSM_PROTOCOL=http://",
		"HSM_HOST_WITH_PORT=cray-smd",
		"HSM_BASE_PATH=/hsm/v1",
		"SLS_PROTOCOL=http://",
		"SLS_HOST_WITH_PATH=cray-sls",
		"CAPMC_PROTOCOL=http://",
		"CAPMC_HOST_WITH_PORT=cray-capmc",
		"CAPMC_BASE_PATH=/capmc/v1",
		"SLEEP_LENGTH=30",
		"FEATURE_FLAG_SLS=False",
	)
	output, err := command.CombinedOutput()

	mountainLogger := logger.With(zap.String("source", "mountain_helper"))
	for _, line := range strings.Split(strings.TrimSuffix(string(output), "\n"), "\n") {
		// This is pretty dang nerdy, but use regex to parse each line and get its equivalent logging level and message.
		loggingMatches := mountainLoggingRegex.FindStringSubmatch(line)
		if len(loggingMatches) == 3 {
			level := loggingMatches[1]
			message := loggingMatches[2]

			switch strings.ToLower(level) {
			case "debug":
				mountainLogger.Debug(message)
			case "info":
				mountainLogger.Info(message)
			case "error":
				mountainLogger.Error(message)
			default:
				mountainLogger.Info(message)
			}
		} else {
			mountainLogger.Info(line)
		}
	}

	if err != nil {
		logger.Error("Mountain discovery script failed to exec!", zap.Error(err))
	} else {
		logger.Info("Mountain discovery finished.")
	}
}
