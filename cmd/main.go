package main

import (
	"encoding/hex"
	"fmt"
)

func main() {
	expectedHex := "70edcac2611a8829ebf467a6849f5d8408d9d8f4"
	gotHex := "9f550f341ed660547cc9789846b62c83b8013427"

	expectedBytes, err := hex.DecodeString(expectedHex)
	if err != nil {
		panic(err)
	}
	gotBytes, err := hex.DecodeString(gotHex)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Expected bytes: %x\n", expectedBytes)
	fmt.Printf("Got bytes: %x\n", gotBytes)
}
