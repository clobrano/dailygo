package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type Task struct {
	Start       string
	Stop        string
	Description string
	Notes       string
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Please provide the path to the planning file as an argument")
	}
	filePath := os.Args[1]

	tasks, err := parsePlanningFile(filePath)
	if err != nil {
		log.Fatalf("Failed to parse planning file: %v", err)
	}

	for _, task := range tasks {
		fmt.Printf("Start: %s\n", task.Start)
		fmt.Printf("Stop: %s\n", task.Stop)
		fmt.Printf("Description: %s\n", task.Description)
		fmt.Printf("Notes: %s\n", task.Notes)
		fmt.Println()

		go scheduleNotification(task)
	}

	select {}
}

func parsePlanningFile(filePath string) ([]Task, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tasks []Task
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		task, err := parseTaskLine(line)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	return tasks, nil
}

func parseTaskLine(line string) (Task, error) {
	var task Task

	re := regexp.MustCompile(`(\d{1,2})(?::(\d{2}))?\s*(?:-\s*(\d{1,2})(?::(\d{2}))?)?\s*;\s*(.*?)(?:\s*;\s*(.*))?$`)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 6 {
		return task, fmt.Errorf("invalid task format: %s", line)
	}

	startHour := matches[1]
	startMin := matches[2]
	stopHour := matches[3]
	stopMin := matches[4]

	task.Start = formatTime(startHour, startMin)
	task.Stop = formatTime(stopHour, stopMin)
	task.Description = parseWikiLink(strings.TrimSpace(matches[5]))

	if len(matches) == 7 {
		task.Notes = strings.TrimSpace(matches[6])
	}

	return task, nil
}

func formatTime(hour, min string) string {
	if min == "" {
		min = "00"
	}
	return fmt.Sprintf("%02s:%02s", hour, min)
}

func parseWikiLink(description string) string {
	re := regexp.MustCompile(`\[\[(.*?)\]\]`)
	return re.ReplaceAllString(description, "$1")
}

func scheduleNotification(t Task) {
	now := time.Now()
	start, err := time.Parse("15:04", t.Start)
	if err != nil {
		log.Printf("Failed to parse start time: %v", err)
		return
	}

	notificationTime := time.Date(now.Year(), now.Month(), now.Day(), start.Hour(), start.Minute(), 0, 0, time.Local)
	if notificationTime.Before(now) {
		notificationTime = notificationTime.AddDate(0, 0, 1)
	}

	// Extract notification duration from notes, if present
	duration := time.Duration(0)
	if t.Notes != "" {
		durationStr := extractNotificationDuration(t.Notes)
		if durationStr != "" {
			duration, err = time.ParseDuration(durationStr)
			if err != nil {
				log.Printf("Failed to parse notification duration: %v", err)
			}
		}
	}

	// Subtract notification duration from the notification time
	notificationTime = notificationTime.Add(-duration)

	sleepDuration := notificationTime.Sub(now)
	time.Sleep(sleepDuration)

	// Execute notify-send command to show notification
	cmd := exec.Command("notify-send", "Task Starting", fmt.Sprintf("Time: %s\nDescription: %s", t.Start, t.Description))
	err = cmd.Run()
	if err != nil {
		log.Printf("Failed to send desktop notification: %v", err)
	}

	fmt.Printf("\n--- Task Starting ---\n")
	fmt.Printf("Time: %s\n", t.Start)
	fmt.Printf("Description: %s\n", t.Description)
	fmt.Printf("---------------------\n")
}

func extractNotificationDuration(notes string) string {
	re := regexp.MustCompile(`!(-\d+)`)
	matches := re.FindStringSubmatch(notes)
	if len(matches) == 2 {
		// Extract the negative duration value (e.g., -5)
		durationStr := matches[1]
		return durationStr + "m" // Append 'm' to indicate minutes
	}
	return ""
}
