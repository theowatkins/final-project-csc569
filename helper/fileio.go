package helper

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

//reads next string, and if "exit" is given stops application
func ReadString(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	text, _ := reader.ReadString('\n')
	cleanText := strings.TrimSpace(text)
	if cleanText == "exit" {
		log.Fatal("exitting program...")
	}
	return cleanText
}

//reads next line, parses into int, and if "exit" is given stops application
func ReadInt(prompt string) int {
	res := ReadString(prompt)
	i, err := strconv.Atoi(res)
	if err != nil {
		log.Fatal("could not parse string into int:", res)
	}
	return i
}

//reads next line, parses into bool, and if "exit" is given stops application
func ReadBool(prompt string) bool {
	res := ReadString(prompt)
	b, err := strconv.ParseBool(res)
	if err != nil {
		log.Fatal("could not parse string into bool:", res)
	}
	return b
}


