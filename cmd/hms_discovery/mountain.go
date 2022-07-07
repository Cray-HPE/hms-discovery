// MIT License
//
// (C) Copyright [2021-2022] Hewlett Packard Enterprise Development LP
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
// OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
// ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

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
https://github.com/Cray-HPE/hms-mountain-discovery

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
		"HSM_BASE_PATH=/hsm/v2",
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
