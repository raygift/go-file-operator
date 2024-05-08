package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

var (
	filepath string
	duration int64
	interval int64
	maxSize  int64 // 文件大小上限，单位MB，日志文件达到上限后会被归档，需要重置offset 并打开新文件
	errCh    chan error
)

func main() {
	var rootCmd = &cobra.Command{
		Use:     "Scanner",
		Short:   "Scanner",
		Version: "Scanner version", // cobra设置--version的固定写法
		Run: func(cmd *cobra.Command, args []string) {
			var fileOffset int64 = 0
			var fileCount int64 = 0
			LoopReadFile(filepath, &fileOffset, &fileCount, duration, interval)
		},
	}

	rootCmd.Flags().StringVarP(&filepath, "filepath", "F", "", "")
	rootCmd.Flags().Int64VarP(&duration, "File Read Duration", "D", 0, "")
	rootCmd.Flags().Int64VarP(&interval, "File Read Interval", "I", 10, "")
	rootCmd.Flags().Int64VarP(&maxSize, "Max Size Per File", "M", 1, "")

	_ = rootCmd.MarkFlagRequired("filepath")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// duration 尝试读取文件的持续时间，单位: 秒（s)
// interval 尝试读取文件内容的间隔，单位: 秒（s)
// offset 读取文件的偏移量
// file 要读取的目标文件名
func LoopReadFile(file string, offset, fileCount *int64, duration, interval int64) error {
	it := time.NewTicker(time.Duration(interval) * time.Second)
	//
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(duration)*time.Second)
	defer cancel()
	for {
		ReadFileContent(file, offset, fileCount)
		select {
		case <-ctx.Done():
			//到达持续时间，退出读取
			return nil
		case msg := <-errCh:
			// 读取到文件末尾，继续尝试读取
			if msg == io.EOF {
				continue
			} else {
				return msg

			}
		case <-it.C:
			continue
			// 到达时间间隔，继续尝试读取
		}

	}
	// return nil
	// data, err := ioutil.ReadFile(filepath.Clean(file))
	// if err != nil {
	// 	return "", err
	// }
	// return string(data), err
}

func ReadFileContent(file string, offset, fileCount *int64) {
	fmt.Println("ReadFileContent offset: ", *offset) // end：读取文件之后的偏移量

	f, err := os.OpenFile(file, os.O_RDWR, os.ModePerm)
	if err != nil {
		log.Fatal(err)
		errCh <- err

	}
	defer f.Close()

	// 设置偏移量
	end, err := f.Seek(*offset, io.SeekCurrent)
	if err != nil {
		log.Fatal(err)
		errCh <- err
	}
	start := time.Now()
	// 读取文件
	bytes, err := ioutil.ReadAll(f)
	finish := time.Since(start)
	if err != nil {
		log.Fatal(err)
		errCh <- err
	}

	// 获取最新的偏移量
	// Seek(offset, whence) 用于设置偏移量， offset 偏移量，whence 偏移量相对位置，
	// io.SeekStart, whence 为0 表示offset 相对于文件起始处，
	// io.SeekCurrent, whence==1 表示 offset 为相对于文件的当前偏移，
	// io.SeekEnd, whence==2 表示offset 为相对于文件结尾处
	end, err = f.Seek(0, io.SeekCurrent)
	if err != nil {
		log.Fatal(err)
		errCh <- err
	}

	// 若读取位置与 offset 相同，说明本次未读取到新内容
	// 本次读取结束
	if *offset == end {
		fmt.Println("no new line in file")
		// 已读取到的文件内容大于文件大小上限
		if *offset >= maxSize*1024*1024 {
			// 当前文件将被归档
			// 重置 offset
			*offset = 0
			*fileCount += 1
			fmt.Println("reset offset")
		}
		// 退出本次读取
		return
	}

	// 否则本地读取到了内容
	// 下面处理本次读取到的内容
	fmt.Printf("read file size(%d)bytes, cost(%d)ms, update offset:%d\n", len(bytes), finish/1e6, *offset)

	// 更新最新的偏移量
	*offset = end
	// fmt.Println("update offset: ", *offset) // end：读取文件之后的偏移量

	// 将读取到的内容写入结果文件
	fileName := path.Base(file)
	pathName := file[0 : len(file)-len(fileName)]
	resultFile := pathName + "result_" + strconv.Itoa(int(*fileCount)) + "_" + fileName
	rf, err := os.OpenFile(resultFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		log.Fatal(err)
		errCh <- err

	}
	defer rf.Close()
	_, err = rf.Write(bytes)
	if err != nil {
		log.Fatal(err)
		errCh <- err

	}
	return
}
