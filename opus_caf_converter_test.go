package caf

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func TestBasicCafEncodingDecoding(t *testing.T) {
	contents, err := os.ReadFile("out_ffmpeg.caf")
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) == 0 {
		t.Fatal("testing with empty file")
	}
	reader := bytes.NewReader(contents)
	f := &File{}
	if err := f.Decode(reader); err != nil {
		t.Fatal(err)
	}
	outputBuffer := &bytes.Buffer{}
	if err := f.Encode(outputBuffer); err != nil {
		t.Fatal(err)
	}
	if outputBuffer.Len() != len(contents) {
		t.Errorf("contents of input differ when decoding and reencoding, before: %d after: %d",
			len(contents),
			outputBuffer.Len())
	}
	output := outputBuffer.Bytes()
	for i := 0; i < len(contents); i++ {
		if output[i] != contents[i] {
			t.Errorf("contents of input differ when decoding and reencoding starting at offset %d", i)
			break
		}
	}
}

func TestConversion(t *testing.T) {
	ConvertOpusToCaf("samples/sample4.opus", "samples/sample4.caf")
}

func TestCompareCaf(t *testing.T) {
	contents1, _ := os.ReadFile("samples/output.caf")
	contents2, _ := os.ReadFile("file.caf")
	if len(contents2) != len(contents1) {
		t.Errorf("contents of input differ when decoding and reencoding, before: %d after: %d",
			len(contents1),
			len(contents2))
	}
	for i := 0; i < len(contents1); i++ {
		if contents2[i] != contents1[i] {
			// print byte content1[i] in fmt.printf as well

			t.Errorf("contents of input differ when decoding and reencoding starting at offset %d %#x %#x", i, contents1[i], contents2[i])
			break
		}
	}
}

func TestOpusDecoding(t *testing.T) {
	input := "samples/sample1.opus"
	opusfile, err := NewFile(input)
	if err != nil {
		fmt.Printf("Could not open file %v\n", err.Error())
	}
	for i := 0; i <= 5; i++ {
		_, err = opusfile.GetSample()
		if err != nil {
			fmt.Printf("GetSingleSample returned Errr %v\n", err.Error())
		}
	}
}

func TestCompareCafFFMpeg(t *testing.T) {
	// specify the input and output files
	inputFile := "samples/sample5.opus"
	outputFileFFmpeg := "out_ffmpeg.caf"
	outputFileCode := "out_code.caf"

	// ffmpeg -i in.opus -c:a copy out.caf
	// run the ffmpeg command to convert the audio file
	cmd := exec.Command("ffmpeg", "-i", inputFile, "-c:a", "copy", outputFileFFmpeg)
	err := cmd.Run()
	if err != nil {
		// handle error
		return
	}

	ConvertOpusToCaf(inputFile, outputFileCode)

	contents1, _ := os.ReadFile(outputFileFFmpeg)
	contents2, _ := os.ReadFile(outputFileCode)

	// conversion successful
	println("Conversion complete")

	if len(contents2) != len(contents1) {
		t.Errorf("contents of input differ when decoding and reencoding, before: %d after: %d",
			len(contents1),
			len(contents2))
	}
	for i := 0; i < len(contents1); i++ {
		if contents2[i] != contents1[i] {
			// print byte content1[i] in fmt.printf as well

			t.Errorf("contents of input differ when decoding and reencoding starting at offset %d %#x %#x", i, contents1[i], contents2[i])
			break
		}
	}

}

// func TestMaina(t *testing.T) {
// 	arr := []int{255, 152, 236, 233, 241, 236, 250, 255, 13, 255, 11, 255, 3, 255, 8, 255, 3, 248, 255, 12, 255, 5, 255, 8, 255, 7, 255, 4, 255, 9, 255, 7, 253, 255, 148, 255, 142, 255, 35, 255, 37, 255, 50, 255, 44, 255, 39, 255, 48, 255, 44, 255, 44, 255, 41, 255, 42, 255, 41, 255, 104, 255, 45, 255}
// 	newArr := make([]int, 0)

// 	for i := 0; i < len(arr); i++ {
// 		if i < len(arr) && arr[i] == 255 {
// 			sum := arr[i]
// 			i++
// 			for i < len(arr) && arr[i] == 255 {
// 				sum += arr[i]
// 				i++
// 			}
// 			if i < len(arr) {
// 				sum += arr[i]
// 			}

// 			newArr = append(newArr, sum)
// 		} else {
// 			newArr = append(newArr, arr[i])
// 		}
// 	}

// 	fmt.Println(newArr)
// }
