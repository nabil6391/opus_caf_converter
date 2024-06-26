#!/bin/bash

# Directory containing the .opus files
input_directory="./caf/samples"
output_directory="./caf/ffmpeg"

# Create the output directory if it does not exist
mkdir -p "$output_directory"

# Loop through all .opus files in the input directory
for input_file in "$input_directory"/*.opus; do
  # Get the base name of the file (without the directory and extension)
  base_name=$(basename "$input_file" .opus)
  
  # Set the output file path
  output_file="$output_directory/$base_name.caf"
  
  # Convert the file
  ffmpeg -i "$input_file" -c copy "$output_file"
done