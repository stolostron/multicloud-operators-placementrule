// Copyright 2019 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"regexp"
	"runtime"
)

// QuiteLogLel defines log level for quite
const QuiteLogLel = 4

// NoiseLogLel defines log level for some details
const NoiseLogLel = 5

// VeryNoisy defines log level for all details
const VeryNoisy = 10

var regexStripFnPreamble = regexp.MustCompile(`^.*\.(.*)$`)

// GetFnName get the name of function
func GetFnName() string {
	fnName := "<unknown>"
	// Skip this function, and fetch the PC and file for its parent
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		fnName = regexStripFnPreamble.ReplaceAllString(runtime.FuncForPC(pc).Name(), "$1")
	}
	return fnName
}

// EnterFnString return common string for entering a function
func EnterFnString() string {
	return "Entering " + GetFnName()
}

// ExitFuString returns commong string for leaving a function
func ExitFuString() string {
	return "Leaving " + GetFnName()
}
