// Copyright 2015 The kingshard Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"

	"github.com/flike/kingshard/config"
	"github.com/flike/kingshard/core/golog"
	"github.com/flike/kingshard/proxy/server"
)

var configFile *string = flag.String("config", "/etc/kingshard.conf", "kingshard config file")
var logLevel *string = flag.String("log-level", "", "log level [debug|info|warn|error], default error")

const (
	sqlLogName = "sql.log"
	sysLogName = "sys.log"
	MaxLogSize = 1024 * 1024 * 1024
)

const banner string = `
    __   _                  __                   __
   / /__(_)___  ____ ______/ /_  ____ __________/ /
  / //_/ / __ \/ __ \/ ___/ __ \ / __\/ ___/ __  /
 / ,< / / / / / /_/ (__  ) / / / /_/ / /  / /_/ /
/_/|_/_/_/ /_/\__, /____/_/ /_/\__,_/_/   \__,_/
             /____/
`

func main() {
	fmt.Print(banner)
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	if len(*configFile) == 0 {
		fmt.Println("must use a config file")
		return
	}

	cfg, err := config.ParseConfigFile(*configFile)
	if err != nil {
		fmt.Printf("parse config file error:%v\n", err.Error())
		return
	}

	//when the log file size greater than 1GB, kingshard will generate a new file
	if len(cfg.LogPath) != 0 {
		sysFilePath := path.Join(cfg.LogPath, sysLogName)
		sysFile, err := golog.NewRotatingFileHandler(sysFilePath, MaxLogSize, 1)
		if err != nil {
			fmt.Printf("new log file error:%v\n", err.Error())
			return
		}
		golog.GlobalSysLogger = golog.New(sysFile, golog.Lfile|golog.Ltime|golog.Llevel)

		sqlFilePath := path.Join(cfg.LogPath, sqlLogName)
		sqlFile, err := golog.NewRotatingFileHandler(sqlFilePath, MaxLogSize, 1)
		if err != nil {
			fmt.Printf("new log file error:%v\n", err.Error())
			return
		}
		golog.GlobalSqlLogger = golog.New(sqlFile, golog.Lfile|golog.Ltime|golog.Llevel)
	}

	if *logLevel != "" {
		setLogLevel(*logLevel)
	} else {
		setLogLevel(cfg.LogLevel)
	}

	var svr *server.Server
	svr, err = server.NewServer(cfg)
	if err != nil {
		golog.Error("main", "main", err.Error(), 0)
		golog.GlobalSysLogger.Close()
		golog.GlobalSqlLogger.Close()
		return
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		sig := <-sc
		golog.Info("main", "main", "Got signal", 0, "signal", sig)
		golog.GlobalSysLogger.Close()
		golog.GlobalSqlLogger.Close()
		svr.Close()
	}()

	svr.Run()
}

func setLogLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		golog.GlobalSysLogger.SetLevel(golog.LevelDebug)
	case "info":
		golog.GlobalSysLogger.SetLevel(golog.LevelInfo)
	case "warn":
		golog.GlobalSysLogger.SetLevel(golog.LevelWarn)
	case "error":
		golog.GlobalSysLogger.SetLevel(golog.LevelError)
	default:
		golog.GlobalSysLogger.SetLevel(golog.LevelError)
	}
}
