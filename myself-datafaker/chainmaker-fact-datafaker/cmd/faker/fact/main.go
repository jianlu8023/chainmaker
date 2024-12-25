package main

import (
	"fmt"
	"time"

	"chainmaker-fact-datafaker/internal/logger"
	"chainmaker-fact-datafaker/internal/task"
)

func main() {
	fmt.Println("hello world")

	// flag.String()
	time.Sleep(5 * time.Second)
	secondFunc := func() {
		logger.GetAppLogger().Infof("hello world every secend")
	}

	addTask, err := task.AddTask(secondFunc, "0/1 * * * * * ")
	if err != nil {
		fmt.Println("add task error ", err)
		task.Close()
		return
	}

	fmt.Println("add task success ", addTask)

	minuteId, err := task.AddTask(func() {
		fmt.Println("hello world every    minute ")
	}, "0 */1 * * * *")
	if err != nil {
		fmt.Println("add task error ", err)
		task.Close()
		return
	}

	fmt.Println("add task success ", minuteId)

	time.Sleep(2 * time.Minute)

	fmt.Println("start remove task")

	task.RemoveTask(addTask)

	fmt.Println("remove task success")

	time.Sleep(1 * time.Minute)

	fmt.Println("close task")
	time.Sleep(2 * time.Minute)
	task.Close()

	select {}

}
