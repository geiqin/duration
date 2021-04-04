package main

import (
	"fmt"
	"github.com/geiqin/duration/audio/mp3"
	"log"
	"math"
	"os"
	"time"
)

//获取秒部分时间
func GetSeconds(dur time.Duration) int64 {
	t := math.Ceil(dur.Seconds())
	return int64(t)
}

func main() {

	f, err := os.Open("files/slayer.mp3")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()
	dur, err := mp3.NewDecoder(f).Duration()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("sssss aa:", GetSeconds(dur))
	fmt.Printf("%s duration: %s\n", f.Name(), dur)
	// Output: fixtures/kick.wav duration: 204.172335ms

}
