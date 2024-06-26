package caf

import (
	"bytes"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBasicCafEncodingDecoding(t *testing.T) {
	startTime := time.Now()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	startAlloc := m.Alloc

	contents, err := os.ReadFile("samples/sample_large.caf")
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) == 0 {
		t.Fatal("testing with empty file")
	}
	reader := bytes.NewReader(contents)
	f := &CAFFileData{}
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

	duration := time.Since(startTime)
	runtime.ReadMemStats(&m)
	allocatedMemory := float64(m.Alloc-startAlloc) / (1024 * 1024) // Convert to MB

	t.Logf("Test duration: %v", duration)
	t.Logf("Allocated memory: %.2f MB", allocatedMemory)
}

func TestCompareCafFFMpeg(t *testing.T) {
	startTime := time.Now()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	startAlloc := m.Alloc

	inputFile := "samples/sample_large.opus"
	outputFileFFmpeg := "samples/sample_large.caf"
	outputFileCode := "output_large_sample.caf"

	defer os.Remove(outputFileCode) // Clean up file after test

	err := ConvertOpusToCaf(inputFile, outputFileCode)
	require.NoError(t, err)

	contents1, err := os.ReadFile(outputFileFFmpeg)
	require.NoError(t, err)
	contents2, err := os.ReadFile(outputFileCode)
	require.NoError(t, err)

	require.Equal(t, len(contents1), len(contents2), "File sizes differ")
	require.Equal(t, contents1, contents2, "File contents differ")

	duration := time.Since(startTime)
	runtime.ReadMemStats(&m)
	allocatedMemory := float64(m.Alloc-startAlloc) / (1024 * 1024) // Convert to MB

	t.Logf("Test duration: %v", duration)
	t.Logf("Allocated memory: %.2f MB", allocatedMemory)
}

func TestConversionWithDifferentOptions(t *testing.T) {
	testCases := []struct {
		name      string
		inputFile string
	}{
		{"tiny_no_sound", "samples/tiny.opus"},
		{"48kHz", "samples/sample_mono_48000.opus"},
		{"stereo", "samples/sample_stereo.opus"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			startTime := time.Now()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			startAlloc := m.Alloc

			outputFile := "output_" + tc.name + ".caf"
			defer os.Remove(outputFile) // Clean up file after test

			err := ConvertOpusToCaf(tc.inputFile, outputFile)
			require.NoError(t, err)

			// Verify the output file exists and has content
			outputStats, err := os.Stat(outputFile)
			require.NoError(t, err)
			require.True(t, outputStats.Size() > 0, "Output file is empty")

			// TODO: Add more specific checks for sample rate conversion if needed

			duration := time.Since(startTime)
			runtime.ReadMemStats(&m)
			allocatedMemory := float64(m.Alloc-startAlloc) / (1024 * 1024) // Convert to MB

			t.Logf("Test duration: %v", duration)
			t.Logf("Allocated memory: %.2f MB", allocatedMemory)
		})
	}
}

func TestConversionWithDifferentChannels(t *testing.T) {
	testCases := []struct {
		name      string
		inputFile string
		channels  int
	}{
		{"Mono", "samples/sample_mono_48000.opus", 1},
		{"Stereo", "samples/sample_stereo.opus", 2},
		// {"5.1", "samples/sample_5_1.opus", 6},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			startTime := time.Now()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			startAlloc := m.Alloc

			outputFile := "output_" + tc.name + ".caf"
			defer os.Remove(outputFile) // Clean up file after test

			err := ConvertOpusToCaf(tc.inputFile, outputFile)
			require.NoError(t, err)

			// Verify the output file exists and has content
			outputStats, err := os.Stat(outputFile)
			require.NoError(t, err)
			require.True(t, outputStats.Size() > 0, "Output file is empty")

			// TODO: Add checks to verify the number of channels in the output file

			duration := time.Since(startTime)
			runtime.ReadMemStats(&m)
			allocatedMemory := float64(m.Alloc-startAlloc) / (1024 * 1024) // Convert to MB

			t.Logf("Test duration: %v", duration)
			t.Logf("Allocated memory: %.2f MB", allocatedMemory)
		})
	}
}

func TestConversionWithLargeFile(t *testing.T) {
	startTime := time.Now()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	startAlloc := m.Alloc

	inputFile := "samples/sample_large.opus"
	outputFile := "output_sample_large.caf"
	defer os.Remove(outputFile) // Clean up file after test

	err := ConvertOpusToCaf(inputFile, outputFile)
	require.NoError(t, err)

	// Verify the output file exists and has content
	outputStats, err := os.Stat(outputFile)
	require.NoError(t, err)
	require.True(t, outputStats.Size() > 0, "Output file is empty")

	duration := time.Since(startTime)
	runtime.ReadMemStats(&m)
	allocatedMemory := float64(m.Alloc-startAlloc) / (1024 * 1024) // Convert to MB

	t.Logf("Test duration: %v", duration)
	t.Logf("Allocated memory: %.2f MB", allocatedMemory)
}

func TestInvalidInputFile(t *testing.T) {
	startTime := time.Now()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	startAlloc := m.Alloc

	err := ConvertOpusToCaf("non_existent_file.opus", "output.caf")
	require.Error(t, err)

	duration := time.Since(startTime)
	runtime.ReadMemStats(&m)
	allocatedMemory := float64(m.Alloc-startAlloc) / (1024 * 1024) // Convert to MB

	t.Logf("Test duration: %v", duration)
	t.Logf("Allocated memory: %.2f MB", allocatedMemory)
}
