/**
 * Copyright 2015 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// A simple proxy service to forward JSON events and transform or filter them along the way.
package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	. "github.com/Comcast/eel/eel/eellib"
	. "github.com/Comcast/eel/eel/util"
)

func eelCmd(in, tf string, istbe bool) {
	//Gctx = NewDefaultContext(L_InfoLevel)
	Gctx = NewDefaultContext(L_NilLevel)
	stats := new(ServiceStats)
	Gctx.AddValue(EelTotalStats, stats)
	var settings EelSettings
	Gctx.AddConfigValue(EelConfig, &settings)
	InitHttpTransport(Gctx)
	if strings.HasPrefix(tf, "@") {
		buf, err := ioutil.ReadFile(tf[1:])
		if err != nil {
			panic(err)
		}
		tf = string(buf)
	}
	if tf == "" {
		fmt.Printf("blank transformation\n")
		os.Exit(1)
	}
	if strings.HasPrefix(in, "@") {
		buf, err := ioutil.ReadFile(in[1:])
		if err != nil {
			fmt.Printf("bad in file\n")
			os.Exit(1)
		}
		in = string(buf)
	}
	if in != "" {
		out, errs := EELSimpleTransform(Gctx, in, tf, istbe)
		if errs != nil {
			fmt.Printf("bad transformation %s on %s\n", tf, in)
			for _, e := range errs {
				fmt.Printf("%s\n", e.Error())
			}
			os.Exit(1)
		}
		_, err := os.Stdout.WriteString(out + "\n")
		if err != nil {
			fmt.Printf("cannot write to stdout\n")
			os.Exit(1)
		}
		os.Exit(0)
	}
	stdin := bufio.NewReader(os.Stdin)
	for {
		in, err := stdin.ReadString('\n')
		if err != nil {
			fmt.Printf("cannot read from stdin\n")
			os.Exit(1)
		}
		if in != "" {
			out, errs := EELSimpleTransform(Gctx, in, tf, istbe)
			if errs != nil {
				fmt.Printf("bad transformation\n")
				for _, e := range errs {
					fmt.Printf("%s\n", e.Error())
				}
				os.Exit(1)
			}
			if out != "" {
				_, err = os.Stdout.WriteString(out + "\n")
				if err != nil {
					fmt.Printf("cannot write to stdout\n")
					os.Exit(1)
				}
			}
		}
	}
}
