package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	RootCmd = &cobra.Command{
		Use:           "service-generator",
		Short:         "service-generator creates systemd Unit files",
		Long:          "service-generator is a convenient little tool to create systemd Unit files",
		SilenceErrors: false,
		SilenceUsage:  true,
	}
)

type Strings []string

func (s Strings) IndexOf(n string) int {
	for i, v := range s {
		if v == n {
			return i
		}
	}

	return -1
}

func (s Strings) Contains(n string) bool {
	for _, v := range s {
		if v == n {
			return true
		}
	}

	return false
}

func readString(prompt string, required bool) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s: ", prompt)
	text, err := reader.ReadString('\n')
	text = strings.TrimSpace(text)

	if required && len(text) == 0 {
		return "", errors.New("Required string is empty")
	}

	return text, err
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
