package caf

import (
	"os"
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// memoryUsage returns the current memory usage in bytes
func memoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

func runTest(t *testing.T, name string, testFunc func()) {
	runtime.GC()
	time.Sleep(time.Second) // Allow GC to complete

	startTime := time.Now()
	startMemory := memoryUsage()

	// Disable GC
	debug.SetGCPercent(-1)
	defer debug.SetGCPercent(100)

	testFunc()

	endMemory := memoryUsage()
	duration := time.Since(startTime)

	memoryChange := float64(endMemory-startMemory) / (1024 * 1024)

	t.Logf("%s - Duration: %v", name, duration)
	t.Logf("%s - Memory change: %.2f MB", name, memoryChange)
}

func TestBasicCafEncodingDecoding(t *testing.T) {
	runTest(t, "TestBasicCafEncodingDecoding", func() {
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
	})
}

func TestCompareCafFFMpeg(t *testing.T) {

	testCases := []struct {
		name       string
		inputFile  string
		outputFile string
	}{
		{"tiny", "samples/tiny.opus", "ffmpeg/tiny.caf"},
		{"sample_mono_48000", "samples/sample_mono_48000.opus", "ffmpeg/sample_mono_48000.caf"},
		{"sample_stereo", "samples/sample_stereo.opus", "ffmpeg/sample_stereo.caf"},
		{"sample_large", "samples/sample_large.opus", "ffmpeg/sample_large.caf"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, "TestCompareCafFFMpeg", func() {
				inputFile := tc.inputFile
				outputFileFFmpeg := tc.outputFile
				outputFileCode := "output_" + tc.name + ".caf"

				// defer os.Remove(outputFileCode) // Clean up file after test

				err := ConvertOpusToCaf(inputFile, outputFileCode)
				require.NoError(t, err)

				contents1, err := os.ReadFile(outputFileFFmpeg)
				require.NoError(t, err)
				contents2, err := os.ReadFile(outputFileCode)
				require.NoError(t, err)

				require.Equal(t, len(contents1), len(contents2), "File sizes differ")
				if len(contents1) == len(contents2) {
					for i := range contents1 {
						if contents1[i] != contents2[i] {
							t.Errorf("File contents differ at byte index %d: file1=%d, file2=%d", i, contents1[i], contents2[i])
							break
						}
					}
				} else {
					require.Equal(t, contents1, contents2, "File contents differ")
				}

			})
		})
	}
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
			runTest(t, "TestConversionWithDifferentOptions/"+tc.name, func() {
				outputFile := "output_" + tc.name + ".caf"
				defer os.Remove(outputFile) // Clean up file after test

				err := ConvertOpusToCaf(tc.inputFile, outputFile)
				require.NoError(t, err)

				// Verify the output file exists and has content
				outputStats, err := os.Stat(outputFile)
				require.NoError(t, err)
				require.True(t, outputStats.Size() > 0, "Output file is empty")
			})
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
			runTest(t, "TestConversionWithDifferentChannels/"+tc.name, func() {
				outputFile := "output_" + tc.name + ".caf"
				defer os.Remove(outputFile) // Clean up file after test

				err := ConvertOpusToCaf(tc.inputFile, outputFile)
				require.NoError(t, err)

				// Verify the output file exists and has content
				outputStats, err := os.Stat(outputFile)
				require.NoError(t, err)
				require.True(t, outputStats.Size() > 0, "Output file is empty")
			})
		})
	}
}

func TestConversionWithLargeFile(t *testing.T) {
	runTest(t, "TestConversionWithLargeFile", func() {
		inputFile := "samples/sample_large.opus"
		outputFile := "output_sample_large.caf"
		defer os.Remove(outputFile) // Clean up file after test

		err := ConvertOpusToCaf(inputFile, outputFile)
		require.NoError(t, err)

		// Verify the output file exists and has content
		outputStats, err := os.Stat(outputFile)
		require.NoError(t, err)
		require.True(t, outputStats.Size() > 0, "Output file is empty")
	})
}

func TestInvalidInputFile(t *testing.T) {
	runTest(t, "TestInvalidInputFile", func() {
		err := ConvertOpusToCaf("non_existent_file.opus", "output.caf")
		require.Error(t, err)
	})
}
