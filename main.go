package main

import (
	"fmt"
	"github.com/geiqin/duration/audio/mp3"
	"log"
	"os"
)

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

	fmt.Printf("%s duration: %s\n", f.Name(), dur)
	// Output: fixtures/kick.wav duration: 204.172335ms

}
