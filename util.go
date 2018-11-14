package util

import (
	"errors"
	"mahonia"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var CmdOutputDecoder mahonia.Decoder
const GBK = "936"
const UTF8 = "65001"

func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}


func ParseDuration(s string) (t time.Duration, err error) {
	unit := s[len(s)-1:]
	var timeUnit int64
	switch unit {
	case "s":
		timeUnit = int64(time.Second)
	case "m":
		timeUnit = int64(time.Minute)
	case "h":
		timeUnit = int64(time.Hour)
	case "d":
		timeUnit = int64(time.Hour * 24)
	default:
		err = errors.New("invalid time unit")
		return t, err
	}

	countStr := s[0 : len(s)-1]
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return t, err
	}
	var temp int64
	temp = int64(count) * timeUnit
	t = time.Duration(temp)
	return t, nil
}


func RunCommandWithRetry(count int, name string, args ...string) (output string, err error) {
	for i := 0; i < count; i++ {
		output, err = RunCommand(name, args...)
		if err == nil {
			return output, err
		}
	}
	return output, err
}


// Run os command and return output
func RunCommand(name string, args ...string) (output string, err error) {
	if CmdOutputDecoder == nil {
		err = getCmdEncode()
		if err != nil {
			return "", nil
		}
	}
	cmd := exec.Command(name, args...)
	b, err := cmd.Output()
	output = CmdOutputDecoder.ConvertString(string(b))
	if err != nil {
		return output, err
	}
	return output, nil
}

// get cmd encode format
func getCmdEncode() error {
	cmd := exec.Command("chcp")
	b, err := cmd.Output()
	if err != nil {
		return err
	}
	if strings.Contains(string(b), GBK) {
		CmdOutputDecoder = mahonia.NewDecoder("gbk")
	} else if strings.Contains(string(b), UTF8) {
		CmdOutputDecoder = mahonia.NewDecoder("utf8")
	} else {
		CmdOutputDecoder = mahonia.NewDecoder("utf8")
	}
	return nil
}

//Value	Description
//0	    No files were copied. No failure was encountered. No files were mismatched. The files already exist in the destination directory; therefore, the copy operation was skipped.
//1	    All files were copied successfully.
//2	    There are some additional files in the destination directory that are not present in the source directory. No files were copied.
//3	    Some files were copied. Additional files were present. No failure was encountered.
//5	    Some files were copied. Some files were mismatched. No failure was encountered.
//6	    Additional files and mismatched files exist. No files were copied and no failures were encountered. This means that the files already exist in the destination directory.
//7	    Files were copied, a file mismatch was present, and additional files were present.
//8	    Several files did not copy.
func DealRobocopyResult(o string, e error) (output string, err error) {
	if e != nil && strings.EqualFold(e.Error(), "exit status 2") {
		return o, nil
	}
	return o, e
}
