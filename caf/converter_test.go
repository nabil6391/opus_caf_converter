package caf

import (
	"bytes"
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
	f := &FileData{}
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
