package common

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

var (
	Port          = flag.Int("port", 3000, "the listening port")
	PrintVersion  = flag.Bool("version", false, "print version and exit")
	PrintHelpFlag = flag.Bool("help", false, "print help and exit")
	LogDir        = flag.String("log-dir", "", "specify the log directory")
	EnableGzip    = flag.Bool("gzip", true, "enable gzip compression")
)

func PrintHelp() {
	fmt.Println("Copyright (C) 2025 Buru. All rights reserved.")
	fmt.Println("GitHub: https://github.com/burugo/one-mcp")
	fmt.Println("Usage: one-mcp [--port <port>] [--log-dir <log directory>] [--version] [--help]")
}

func init() {
	if err := loadConfigFile(); err != nil {
		log.Fatal(err)
	}

	if os.Getenv("SESSION_SECRET") != "" {
		SessionSecret = os.Getenv("SESSION_SECRET")
	}
	if os.Getenv("SQLITE_PATH") != "" {
		SQLitePath = os.Getenv("SQLITE_PATH")
	} else {
		// check if the directory exists
		if _, err := os.Stat(filepath.Dir(SQLitePath)); os.IsNotExist(err) {
			err = os.MkdirAll(filepath.Dir(SQLitePath), 0755)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	if os.Getenv("JWT_SECRET") != "" {
		JWTSecret = os.Getenv("JWT_SECRET")
	}
	if os.Getenv("JWT_REFRESH_SECRET") != "" {
		JWTRefreshSecret = os.Getenv("JWT_REFRESH_SECRET")
	} else if os.Getenv("JWT_SECRET") != "" {
		JWTRefreshSecret = os.Getenv("JWT_SECRET")
	}
	if os.Getenv("PORT") != "" {
		portInt, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			log.Fatal(err)
		}
		*Port = portInt
	}

	if os.Getenv("ENABLE_GZIP") != "" {
		enableGzipBool, err := strconv.ParseBool(os.Getenv("ENABLE_GZIP"))
		if err != nil {
			log.Fatalf("invalid value for ENABLE_GZIP: %v", err)
		}
		*EnableGzip = enableGzipBool
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
}
