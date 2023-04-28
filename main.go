package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/nixargh/yad"
	log "github.com/sirupsen/logrus"
	//	"github.com/pkg/profile"
)

var version string = "0.1.0"

var clog *log.Entry

func main() {
	user, err := user.Current()
	if err != nil {
		fmt.Errorf(err.Error())
	}

	username := user.Username
	tokenFile := fmt.Sprintf("/home/%s/.config/yandex-disk/passwd", username)

	//	defer profile.Start().Stop()

	var jsonLog bool
	var debug bool
	var logCaller bool
	var showVersion bool
	var srcPath string
	var dstPath string
	var overwrite bool

	flag.BoolVar(&jsonLog, "jsonLog", false, "Log in JSON format")
	flag.BoolVar(&debug, "debug", false, "Log debug messages")
	flag.BoolVar(&logCaller, "logCaller", false, "Log message caller (file and line number)")
	flag.BoolVar(&showVersion, "version", false, "Groxy version")
	flag.StringVar(&tokenFile, "tokenFile", tokenFile, "File with OAuth token")
	flag.StringVar(&srcPath, "srcpath", srcPath, "Source file path")
	flag.StringVar(&dstPath, "dstpath", dstPath, "Destination file path")
	flag.BoolVar(&overwrite, "overwrite", false, "Overwrite destination files")

	flag.Parse()

	// Setup logging
	log.SetOutput(os.Stdout)

	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if jsonLog == true {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{
			//	FullTimestamp: true,
		})
	}

	if debug == true {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.SetReportCaller(logCaller)

	clog = log.WithFields(log.Fields{
		"pid":     os.Getpid(),
		"logger":  "backyard",
		"version": version,
	})

	clog.Info("Backyard maintenance begins!")
	errors := 0

	if srcPath == "" {
		clog.Fatal("You must set '-srcpath'.")
	}

	if dstPath == "" {
		clog.Fatal("You must set '-dstpath'.")
	}

	// Start logic here
	oauthToken := readOauthToken(tokenFile)

	api := yad.NewAPI(oauthToken, 10*time.Second, true, clog)

	// Upload single file or recursively list and upload all files from directory
	srcPathFile, err := os.Open(srcPath)
	if err != nil {
		// handle the error and return
	}
	defer srcPathFile.Close()

	srcPathInfo, err := srcPathFile.Stat()
	if srcPathInfo.IsDir() {
		clog.Info("Source path is a directory.")

		inputChan := make(chan [2]string)
		errorChan := make(chan error)

		go api.UploadChannelling(inputChan, errorChan, true)
		go listFiles(srcPath, inputChan)

		sleepSeconds := 1
		clog.WithFields(log.Fields{
			"sleepSeconds": sleepSeconds,
		}).Info("Starting a waiting loop.")

		finished := false
		for !finished {
			select {
			case err, opened := <-errorChan:
				if opened {
					clog.Error(err)
					errors++
				} else {
					clog.Info("Error channel is closed.")
					finished = true
				}
			default:
				time.Sleep(time.Duration(sleepSeconds) * time.Second)
			}
		}
	} else {
		clog.Info("Source path is a file.")
		if api.Upload(srcPath, dstPath, overwrite) == false {
			clog.Fatal("Failed to upload file(s) to Yandex Disk.")
		}
	}

	if errors > 0 {
		clog.WithFields(log.Fields{"errors": errors}).Fatal("Finished with errors.")
	} else {
		clog.Info("Finished successfully.")
		os.Exit(0)
	}
}

func readOauthToken(tokenFile string) string {
	// Read OAuth2 token from file
	clog.Info("Reading OAuth token.")

	absPathToToken, _ := filepath.Abs(tokenFile)
	content, err := ioutil.ReadFile(absPathToToken)

	if err != nil {
		clog.WithFields(log.Fields{"error": err}).Fatal("Failed to read OAuth token.")
	}

	return strings.TrimRight(string(content), "\n")
}

func listFiles(srcDir string, inputChan chan [2]string) {

	pushToChan := func(srcPath string, f os.FileInfo, err error) error {
		if f.Mode().IsRegular() {
			dstPath := strings.ReplaceAll(srcPath, srcDir, "")
			var pair = [2]string{srcPath, dstPath}

			clog.WithFields(log.Fields{
				"srcPath": srcPath,
				"dstPath": dstPath,
			}).Info("Adding pair to the input channel.")
			inputChan <- pair
		}

		return nil
	}

	filepath.Walk(srcDir, pushToChan)
	close(inputChan)
}
