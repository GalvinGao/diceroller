package main

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/justinian/dice"
)

var ErrorIgnore = errors.New("ignore such error")

func roll(desc string) (d string, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in roll", r)
			assertError, ok := r.(error)
			if ok {
				err = errors.New("roll failed (panicked): " + assertError.Error())
			} else {
				err = errors.New("roll failed (panicked): unknown panic reason")
			}
		}
	}()

	split := strings.Split(desc, "d")
	if len(split) != 2 {
		return "", ErrorIgnore
	}

	countStr := split[0]
	if countStr == "" {
		countStr = "1"
	}

	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid dice string: expected number of dice count, but got \"%v\"", split[0])
	}

	headStr := split[1]
	if headStr == "" {
		headStr = "6"
	}

	if count <= 0 || count > 100000 {
		return "", errors.New("invalid dice count range: dice count should in range (0, 100000]")
	}

	rand.Seed(time.Now().UnixNano())

	result, _, err := dice.Roll(desc)
	if err != nil {
		return "", errors.New("roll failed: " + err.Error())
	}

	hist := []string{}

	r := result.(dice.StdResult)
	for _, roll := range r.Rolls {
		hist = append(hist, strconv.FormatInt(int64(roll), 10))
	}

	list := strings.Join(hist, ", ")

	if len(list) > 10000 {
		list = list[:10000] + "..."
	}

	return fmt.Sprintf("r %vd%v: %v (%v)", count, headStr, r.Total, list), nil
}
