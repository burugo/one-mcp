package common

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var (
	Port          = flag.Int("port", 3000, "the listening port")
	PrintVersion  = flag.Bool("version", false, "print version and exit")
	PrintHelpFlag = flag.Bool("help", false, "print help and exit")
	LogDir        = flag.String("log-dir", "", "specify the log directory")
)

// UploadPath Maybe override by ENV_VAR
var UploadPath = "upload"

func PrintHelp() {
	fmt.Println("Copyright (C) 2025 Buru. All rights reserved.")
	fmt.Println("GitHub: https://github.com/burugo/one-mcp")
	fmt.Println("Usage: one-mcp [--port <port>] [--log-dir <log directory>] [--version] [--help]")
}

func init() {
	if os.Getenv("SESSION_SECRET") != "" {
		SessionSecret = os.Getenv("SESSION_SECRET")
	}
	if os.Getenv("SQLITE_PATH") != "" {
		SQLitePath = os.Getenv("SQLITE_PATH")
	}
	if os.Getenv("UPLOAD_PATH") != "" {
		UploadPath = os.Getenv("UPLOAD_PATH")
	}
	if os.Getenv("JWT_SECRET") != "" {
		JWTSecret = os.Getenv("JWT_SECRET")
	}
	if os.Getenv("JWT_REFRESH_SECRET") != "" {
		JWTRefreshSecret = os.Getenv("JWT_REFRESH_SECRET")
	} else if os.Getenv("JWT_SECRET") != "" {
		JWTRefreshSecret = os.Getenv("JWT_SECRET")
	}
	if *LogDir != "" {
		var err error
		*LogDir, err = filepath.Abs(*LogDir)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := os.Stat(*LogDir); os.IsNotExist(err) {
			err = os.Mkdir(*LogDir, 0777)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	if _, err := os.Stat(UploadPath); os.IsNotExist(err) {
		_ = os.Mkdir(UploadPath, 0777)
	}
	InitOptionMap()
}
