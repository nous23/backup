package main
import (
	"glog"
	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
	"os"
	"time"
)

type Config struct {
	BackupDirs []string `yaml:"backupDirs"`
	DstDirs string `yaml:"destination_dirs"`
}

func main() {
	flag.Parse()
	defer glog.Flush()
	glog.Info("start backup")
	config, err := readConfig()
	if err != nil {
		glog.Error(err)
		return
	}

	worker(config)
}

func worker(config *Config) {
	for {
		for _, backupDir := range config.BackupDirs {
			if Exists(backupDir) {
				glog.Info(backupDir + " exists.")
			} else {
				glog.Warning(backupDir + " dose not exist, will ignore.")
			}
		}

		time.Sleep(time.Second * 5)
	}
}

func readConfig() (*Config, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		glog.Error(err)
	}
	configFile := dir + "\\conf\\backup.yaml"
	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	var c Config
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		glog.Fatal("Unmarshal: %v", err)
		return nil, err
	}
	return &c, nil
}

func Exists(path string) bool {
	_, err := os.Stat(path)    //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

