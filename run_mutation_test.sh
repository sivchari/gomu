#!/bin/bash
# Script to run mutation testing on the gomu project itself

echo "Running mutation testing on gomu project..."
echo "This may take a while..."

# Build gomu first
go build -o gomu cmd/gomu/main.go

# Run mutation testing on the project
./gomu run . --workers 4 --timeout 30 --output json --threshold 80

# Show the results
if [ -f mutation-report.json ]; then
    echo -e "\n=== Mutation Testing Report ==="
    jq '.mutationScore' mutation-report.json 2>/dev/null && echo "% mutation score"
    echo -e "\nTop files by mutation score:"
    jq -r '.files | to_entries | sort_by(.value.mutationScore) | reverse | .[:10] | .[] | "\(.value.mutationScore | tostring[0:5])% - \(.key)"' mutation-report.json 2>/dev/null
fi