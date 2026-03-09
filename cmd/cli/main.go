package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Vladroon22/2FA/internal/core"
)

func main() {
	secret := os.Args[1]
	fmt.Printf("Ваш секрет: %s\n", secret)

	for {
		now := time.Now().UTC()
		code, err := core.GenerateOTP(now, secret)
		if err != nil {
			fmt.Println(err)
			continue
		}

		remaining := 30 - (now.Unix() % 30)

		fmt.Printf("\rКод: %s | Осталось: %2d сек", code, remaining)
		time.Sleep(1 * time.Second)
	}

}
