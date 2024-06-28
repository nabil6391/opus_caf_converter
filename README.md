# Opus to CAF Converter

This repository provides a Go package for converting Opus audio files to Core Audio Format (CAF) files. The Opus codec is known for high-quality, low-latency audio compression, while the Core Audio Format is a container format developed by Apple for use with their Core Audio framework.

## Significance

This converter is particularly important because Apple does not natively support the Opus codec in its ecosystem. This limitation necessitates the conversion of Opus files to a format that Apple's platforms can readily use, such as CAF.

### Advantages of Opus

Opus is a highly efficient audio codec that can significantly reduce file sizes compared to other formats:

- Opus files are typically about half the size of MP3 files at equivalent quality
- This size reduction is achieved without compromising audio fidelity

### Efficient Conversion

A key feature of this converter is its ability to perform the Opus to CAF conversion:

- Without relying on external tools like FFmpeg
- While maintaining the original audio quality (lossless conversion)
- Directly in Go, making it easy to integrate into existing Go projects

## Installation

To install the package, use the following command:

```sh
go get github.com/nabil6391/opus_caf_converter
```

## Usage

### As a Package

Import the package in your Go code:

```go
import "github.com/nabil6391/opus_caf_converter/caf"
```

Use the `ConvertOpusToCaf` function:

```go
func main() {
    inputOpus := "path/to/input.opus"
    outputCaf := "path/to/output.caf"
    err := caf.ConvertOpusToCaf(inputOpus, outputCaf)
    if err != nil {
        log.Fatalf("Conversion failed: %v", err)
    }
    fmt.Println("Conversion successful!")
}
```

### As a CLI Tool

You can also use this converter as a command-line tool:

1. Install the CLI tool:

```sh
go install github.com/nabil6391/opus_caf_converter@latest
```

2. Run the converter:

```sh
opus_caf_converter input.opus output.caf
```

## Features

- Supports conversion of Opus files to CAF format
- Handles both mono and stereo audio channels
- Preserves audio quality during conversion (lossless conversion)
- Efficient processing of large files
- No dependency on external tools like FFmpeg

## Implementation Details

The conversion process involves several steps:

1. Parsing the Opus file structure
2. Extracting audio data and metadata
3. Constructing the CAF file with appropriate headers and chunks
4. Writing the converted data to the output file

## Testing

The package includes a comprehensive test suite. Run the tests using:

```sh
go test ./...
```

## Limitations

- Currently supports only mono and stereo audio channels
- Does not support all possible Opus configurations

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- The Opus codec developers
- Apple's Core Audio Format documentation

For more detailed information about the implementation, please refer to the source code and comments within the package.