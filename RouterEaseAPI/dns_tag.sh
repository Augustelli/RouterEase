#!/bin/bash

# Check if required arguments are provided
if [ "$#" -lt 2 ]; then
    echo "Usage: $0 input_file tag1 [tag2 ...] [-o output_file]"
    echo "Example: $0 filters.txt os_linux user_children adult -o tagged_filters.txt"
    exit 1
fi

INPUT_FILE="$1"
shift

TAGS=()
OUTPUT_FILE=""

# Parse arguments
while [ "$#" -gt 0 ]; do
    case "$1" in
        -o)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        *)
            TAGS+=("$1")
            shift
            ;;
    esac
done

# Create default output filename if not specified
if [ -z "$OUTPUT_FILE" ]; then
    OUTPUT_FILE="${INPUT_FILE%.*}_tagged.${INPUT_FILE##*.}"
fi

# Join tags with pipe symbol
JOINED_TAGS=$(IFS="|"; echo "${TAGS[*]}")

# Process the file
{
    while IFS= read -r line; do
        # Skip empty lines or preserve them
        if [[ -z "$line" ]]; then
            echo "$line"
        # Preserve comment lines (starting with # or !)
        elif [[ "$line" == \#* || "$line" == !* ]]; then
            echo "$line"
        # Add tags to domain entries
        else
            echo "$line \$ct_tag=${JOINED_TAGS}"
        fi
    done < "$INPUT_FILE"
} > "$OUTPUT_FILE"

echo "Tagged file created: $OUTPUT_FILE"