package main

import (
	"errors"
	"flag"
	"glog-master"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
	"util"
	"values"
)

var BackupStatusCh chan string

type BackupConfig struct {
	DefaultDst    string `yaml:"default_dst"`
	DefaultPeriod string `yaml:"default_period"`
	Tasks         []Task `yaml:"tasks"`
}

func (bc *BackupConfig) Validate() (err error) {
	for index, task := range bc.Tasks {
		bc.Tasks[index].Src = filepath.Clean(task.Src)
		bc.Tasks[index].Dst = filepath.Clean(task.Dst)

		if strings.EqualFold(task.Src, "") {
			err = errors.New(task.Name + " source path is empty")
			glog.Error(err.Error())
			return err
		}

		if strings.EqualFold(task.Dst, "") {
			if bc.DefaultDst != "" {
				bc.Tasks[index].Dst = bc.DefaultDst
			} else {
				err = errors.New(task.Name + " dst未配置")
				glog.Error(err.Error())
				return err
			}
		}

		if strings.EqualFold(task.PeriodString, "") {
			if bc.DefaultPeriod != "" {
				bc.Tasks[index].PeriodString = bc.DefaultPeriod
			} else {
				err = errors.New(task.Name + " period未配置")
				glog.Error(err.Error())
				return err
			}
		}

		if strings.EqualFold(task.Name, "") {
			bc.Tasks[index].Name = "[" + bc.Tasks[index].Src + "-->" + bc.Tasks[index].Dst + "]"
		}
	}
	return nil
}


type Task struct {
	Src            string `yaml:"src"`
	Dst            string `yaml:"dst"`
	PeriodString   string `yaml:"period"`
	PeriodDuration time.Duration
	Name           string `yaml:"name"`
	ticker         <-chan time.Time
	stopCh         chan string
	LastSuccTime time.Time `yaml:"last_succ_time"`
	RecentResult  []string `yaml:"recent_result"`
}

func (t *Task) check() (err error) {
	if !util.Exists(t.Src) {
		err = errors.New(t.Src + " does not exist, will skip the task " + t.Name)
		glog.Error(err.Error())
		return err
	}

	if !util.Exists(t.Dst) {
		glog.Warning(t.Dst + " does not exist, will create it.")
		if err = os.Mkdir(t.Dst, os.ModePerm); err != nil {
			glog.Error("make directory " + t.Dst + " failed: " + err.Error())
			return err
		}
	}

	fi, err := os.Stat(t.Dst)
	if err != nil {
		glog.Error(err.Error())
		return err
	}
	if fi.Mode().IsRegular() {
		err = errors.New("dst directory " + t.Dst + " is regular file!")
		glog.Error(err)
		return err
	}
	return nil
}


func (t *Task) dealResult(err *error) {
	currTime := time.Now()
	result := ""
	if *err == nil {
		t.LastSuccTime = currTime
		result = "success"
	} else {
		result = "fail"
	}

	record := []string{currTime.Format("2006-01-02 15:04:05") + " " + result}

	if len(t.RecentResult) >= values.RecentRecordCount {
		t.RecentResult = append(record, t.RecentResult[0:values.RecentRecordCount - 1]...)
	} else {
		t.RecentResult = append(record, t.RecentResult...)
	}

	BackupStatusCh <- "update status"
}


func (t *Task) work() (err error) {
	defer t.dealResult(&err)
	glog.Infof("start work for task %v", t.Name)
	if err = t.check(); err != nil {
		glog.Error("task check error: " + err.Error() + ", task name: " + t.Name)
		return err
	}

	fi, err := os.Stat(t.Src)
	if err != nil {
		glog.Error(err)
		return err
	}

	// if src is a regular file, just copy it to dst
	var output string
	if fi.Mode().IsRegular() {
		srcFileDir := filepath.Dir(t.Src)
		srcFile := filepath.Base(t.Src)
		output, err = util.DealRobocopyResult(util.RunCommandWithRetry(values.RobocopyRetryCount,"robocopy", srcFileDir, t.Dst, srcFile))
		if err != nil {
			glog.Errorf("exec command {%s} failed: %v\n%v", strings.Join([]string{"robocopy", srcFileDir, t.Dst,
				srcFile}, " "), err, output)
			return err
		}
	} else if fi.Mode().IsDir() {
		dstPath := filepath.Join(t.Dst, filepath.Base(t.Src))
		output, err = util.DealRobocopyResult(util.RunCommandWithRetry(values.RobocopyRetryCount, "robocopy", t.Src, dstPath, "/e"))
		if err != nil {
			glog.Errorf("exec command {%s} failed: %v\n%v", strings.Join([]string{"robocopy", t.Src, dstPath,
				"/e"}, " "), err, output)
			return err
		}
	} else {
		err = errors.New(t.Src + "is neither a file nor a directory.")
		glog.Error(err.Error())
		return err
	}
	glog.V(3).Infof("exec robocopy: %v", output)

	return nil
}

func (t *Task) start() {
	glog.Infof("start task %v", t.Name)
	var interval = time.Now().Sub(t.LastSuccTime)

	if interval > t.PeriodDuration {
		glog.Warningf("task %v has not been executed for %v, which is logger than %v, will execute it right now.",
			t.Name, interval, t.PeriodDuration)
		if err := t.work(); err != nil {
			glog.Error(err.Error())
		}
	} else {
		firstWait := make(chan string, 1)
		go func() {
			time.Sleep(t.PeriodDuration - interval)
			firstWait <- "now"
		}()

		select {
		case <- t.stopCh:
			glog.Warning("task " + t.Name + " stopped.")
			return
		case <- firstWait:
			err := t.work()
			if err != nil {
				glog.Error(err.Error())
			}
		}
	}

	t.ticker = time.Tick(t.PeriodDuration)
	for {
		select {
		case <-t.stopCh:
			glog.Warning("task " + t.Name + " stopped.")
			return
		case <-t.ticker:
			err := t.work()
			if err != nil {
				glog.Error(err.Error())
				continue
			}
		}
	}
}

func (t *Task) equals(task Task) bool {
	if strings.EqualFold(filepath.Clean(t.Src), filepath.Clean(task.Src)) &&
		strings.EqualFold(filepath.Clean(t.Dst), filepath.Clean(task.Dst)) {
			return true
	}
	return false
}

type Config struct {
	//indicate backup.yaml file update
	updateConfigFile chan string
	backupConfig     BackupConfig
	updateTime       time.Time
	configFilePath   string
	//indicate backupConfig struct update
	updateBackupConfig chan string
	statusFilePath string
}

func (c *Config) Init() error {
	u, err := user.Current()
	if err != nil {
		glog.Errorf("get current user failed: %v", err)
		return err
	}
	c.configFilePath = filepath.Join(u.HomeDir, "Documents", "backup", "backup.yaml")
	c.statusFilePath = filepath.Join(u.HomeDir, "Documents", "backup", "backup_status.yaml")
	c.updateTime = time.Time{}
	c.updateConfigFile = make(chan string, 10)
	c.updateBackupConfig = make(chan string, 10)
	c.backupConfig = BackupConfig{}
	return nil
}

func (c *Config) Parse() error {
	yamlFile, err := ioutil.ReadFile(c.configFilePath)
	if err != nil {
		glog.Error(err.Error())
		return err
	}
	var bc BackupConfig
	err = yaml.Unmarshal(yamlFile, &bc)
	if err != nil {
		glog.Error(err.Error())
		return err
	}

	if err = bc.Validate(); err != nil {
		glog.Error("Validate backup config failed: ", err.Error())
		return err
	}

	for index := range bc.Tasks {
		if duration, err := util.ParseDuration(bc.Tasks[index].PeriodString); err != nil {
			glog.Error(err.Error())
			return err
		} else {
			bc.Tasks[index].PeriodDuration = duration
		}
	}

	if util.Exists(c.statusFilePath) {
		var bs BackupConfig
		statusFile, err := ioutil.ReadFile(c.statusFilePath)
		if err != nil {
			glog.Error(err.Error())
			return err
		}
		err = yaml.Unmarshal(statusFile, &bs)
		if err != nil {
			glog.Error(err.Error())
			return err
		}
		for i := range bc.Tasks {
			for j := range bs.Tasks {
				if bc.Tasks[i].equals(bs.Tasks[j]) {
					bc.Tasks[i].LastSuccTime = bs.Tasks[j].LastSuccTime
					bc.Tasks[i].RecentResult = bs.Tasks[j].RecentResult
				}
			}
		}
	}

	c.backupConfig = bc
	c.updateBackupConfig <- "updated"
	glog.Warning("updateBackupConfig signal send")
	return nil
}

func (c *Config) Monit() {
	glog.Info("Start monit backup config...")
	for {
		fileInfo, err := os.Stat(c.configFilePath)
		if err != nil {
			glog.Error(err)
		}
		if fileInfo.ModTime().After(c.updateTime) {
			c.updateTime = fileInfo.ModTime()
			c.updateConfigFile <- "updated"
			glog.Warning("backup config updated at ", c.updateTime, " will send signal.")
		}
		time.Sleep(values.MonitConfigPeriod)
	}
}

func (c *Config) Update() {
	glog.Info("Start updateConfigFile backup config...")
	for {
		select {
		case <-c.updateConfigFile:
			glog.Warning("receive updateConfigFile backup config signal.")
			if err := c.Parse(); err != nil {
				glog.Errorf("parse backup config error: %v, will continue use old config: %+v",
					err.Error(), c.backupConfig)
			}
		case <- BackupStatusCh:
			glog.V(3).Info("receive backup status update signal")
			if err := c.UpdateStatus(); err != nil {
				glog.Error("UpdateStatus error: ", err.Error())
			}
		}
	}
}

func (c *Config) UpdateStatus() error {
	data, err := yaml.Marshal(c.backupConfig)
	if err != nil {
		glog.Error(err)
	}
	err = ioutil.WriteFile(c.statusFilePath, data, os.ModePerm)
	if err != nil {
		glog.Error(err)
	}
	return nil
}




func main() {
	flag.Parse()
	defer glog.Flush()
	glog.Info("start backup process")

	BackupStatusCh = make(chan string, 100)

	var c Config
	if err := c.Init(); err != nil {
		glog.Fatal("init config error: ", err.Error())
		return
	}

	go c.Monit()
	go c.Update()

	mainLoop(&c)
}

func mainLoop(c *Config) {
	glog.Info("Start main loop...")
	for {
		var tasks []Task
		tasks = c.backupConfig.Tasks
		for index := range tasks {
			tasks[index].stopCh = make(chan string, 1)
			go tasks[index].start()
		}

		select {
		case <-c.updateBackupConfig:
			glog.Warning("backup config updated, will restart all tasks.")
			for index := range tasks {
				tasks[index].stopCh <- "stop"
			}
		}
	}
}



