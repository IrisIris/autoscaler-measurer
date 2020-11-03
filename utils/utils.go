package utils

import (
	"fmt"
	"time"
)

func WaitForResultWithError(name string, predicate func() (bool, error), returnError bool, delay int, timeout int) error {
	endTime := time.Now().Add(time.Duration(timeout) * time.Second)
	delaySecond := time.Duration(delay) * time.Second
	for {
		// Sleep
		time.Sleep(delaySecond)
		// If a timeout is set, and that's been exceeded, shut it down
		if timeout >= 0 && time.Now().After(endTime) {
			return fmt.Errorf(fmt.Sprintf("Wait for %s timeout", name))
		}
		// Execute the function
		satisfied, err := predicate()
		if err != nil && !satisfied {
			//log.Warnf("%s Invoke func %++s error %++v", name, "predicate func() (bool, error)", err)
			if returnError {
				return err
			} else {
				continue
			}
		}
		if satisfied {
			return err
		}
	}
}
