package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"util"
	"values"
)

func main() {
	var installPath, logPath, logLevel string
	//fmt.Print("Enter text: ")
	//text, _ := reader.ReadString('\n')
	//fmt.Println(text)
	fmt.Printf("please input install path: ")
	fmt.Scanf("%s", &installPath)
	//installPath, _ = reader.ReadString('\n')
	//installPath = filepath.Clean(installPath)
	//fmt.Printf("installpath: %s", installPath)
	logPath = filepath.Clean(filepath.Join(installPath, "logs"))
	//for {
	//	fmt.Printf("\nuse default log path [%s] (y/n)?", logPath)
	//	useDefaultLogPath, _ = reader.ReadString('\n')
	//	if strings.EqualFold(strings.ToLower(strings.TrimSpace(useDefaultLogPath)), "n") {
	//		fmt.Printf("please input your log path:")
	//		logPath, _ = reader.ReadString('\n')
	//		logPath = filepath.Clean(logPath)
	//		break
	//	} else {
	//		if !strings.EqualFold(strings.ToLower(strings.TrimSpace(useDefaultLogPath)), "y") {
	//			fmt.Println("please input y or n")
	//			continue
	//		}
	//		break
	//	}
	//}
	logLevel = "0"
	//for {
	//	fmt.Printf("use default log level [%s] (y/n)?", logLevel)
	//	useDefaultLogLevel, _ = reader.ReadString('\n')
	//	if strings.EqualFold(strings.ToLower(strings.TrimSpace(useDefaultLogLevel)), "n") {
	//		fmt.Printf("please input your log level (0 ~ 5):")
	//		logLevel, _ = reader.ReadString('\n')
	//		logLevel = strings.TrimSpace(logLevel)
	//		break
	//	} else {
	//		if !strings.EqualFold(strings.ToLower(strings.TrimSpace(useDefaultLogLevel)), "y") {
	//			fmt.Println("please input y or n")
	//			continue
	//		}
	//		break
	//	}
	//}

	output, err := util.RunCommand("taskkill", "/IM", "backup.exe")
	if err != nil {
		fmt.Printf("kill backup failed: %v, \n%v", err, output)
	}

	currDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Printf("get current dir failed: %v", err)
		return
	}
	_, err = util.RunCommandWithRetry(values.RobocopyRetryCount,"robocopy", currDir, filepath.Join(installPath, "backup"), "/e")
	if err != nil {
		fmt.Printf("can't copy program file to install path: %v", err)
		return
	}

	if !util.Exists(logPath) {
		if err = os.Mkdir(logPath, os.ModePerm); err != nil {
			fmt.Printf("make dir %s failed: %v", logPath, err)
			return
		}
	}

	u, err := user.Current()
	if err != nil {
		fmt.Printf("get current user failed: %v", err)
		return
	}

	documentFilePath := filepath.Join(u.HomeDir, "Documents", "backup")
	if !util.Exists(documentFilePath) {
		if err = os.Mkdir(documentFilePath, os.ModePerm); err != nil {
			fmt.Printf("make dir %s failed: %v", documentFilePath, err)
			return
		}
	}

	backupConfigFile := filepath.Join(documentFilePath, "backup.yaml")
	if !util.Exists(backupConfigFile) {
		if output, err := util.RunCommandWithRetry(values.RobocopyRetryCount, "robocopy", filepath.Join(currDir, "conf"), documentFilePath); err != nil {
			fmt.Printf("cp config files to %s failed: %s", documentFilePath, output)
			return
		}
	}

	b, err := ioutil.ReadFile(filepath.Join(currDir, "scripts", "startup_backup_template.vbs"))
	if err != nil {
		fmt.Printf("read file startup_backup_template.vbs failed: %v", err)
		return
	}
	configMap := make(map[string]string)
	configMap["PROGRAM_PATH"] = filepath.Join(installPath, "backup", "backup.exe")
	configMap["LOG_LEVEL"] = logLevel
	configMap["LOG_DIR"] = logPath
	template := string(b)
	for k, v := range configMap {
		template = strings.Replace(template, k, v, 1)
	}

	startupFilePath := filepath.Join(u.HomeDir, "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs",
		"Startup", "start_backup.vbs")
	// write the whole body at once
	err = ioutil.WriteFile(startupFilePath, []byte(template), os.ModePerm)
	if err != nil {
		fmt.Printf("write file to %s failed: %v", startupFilePath, err)
		return
	}

	fmt.Println("Install success.")
}
