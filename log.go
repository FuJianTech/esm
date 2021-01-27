/*
Copyright 2016 Medcl (m AT medcl.net)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	log "github.com/cihub/seelog"
	"strings"
)

func setInitLogging(logLevel string) {

	logLevel = strings.ToLower(logLevel)

	testConfig := `
	<seelog  type="sync" minlevel="`
	testConfig = testConfig + logLevel
	testConfig = testConfig + `">
		<outputs formatid="main">
			<filter levels="error">
				<file path="./esm.log"/>
			</filter>
			<console formatid="main" />
		</outputs>
		<formats>
			<format id="main" format="[%Date(01-02) %Time] [%LEV] [%File:%Line,%FuncShort] %Msg%n"/>
		</formats>
	</seelog>`

	logger, err := log.LoggerFromConfigAsString(testConfig)
	if err != nil {
		log.Error("init config error,", err)
	}
	err = log.ReplaceLogger(logger)
	if err != nil {
		log.Error("init config error,", err)
	}
}
