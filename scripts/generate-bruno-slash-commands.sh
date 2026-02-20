#!/bin/bash

# Directory to save Bruno files
BRUNO_DIR="/mnt/c/Users/USER/Documents/bruno-collections/zedu-be-swagger/Slash-Commands"

# Create directory if it doesn't exist
mkdir -p "$BRUNO_DIR"

# Array of slash commands with examples
declare -A commands=(
  ["add-to-channel"]="/add-to-channel #engineering-channel @bugsbouncy @anotheruser"
  ["remove-from-channel"]="/remove-from-channel #engineering-channel @bugsbouncy"
  ["banish-from-channel"]="/banish-from-channel #general #engineering-channel @bugsbouncy @anotheruser"
  ["restore-channels"]="/restore-channels @bugsbouncy @anotheruser"
  ["export-members"]="/export-members #engineering-channel"
  ["promote"]="/promote #management @bugsbouncy @anotheruser #engineering-channel #support"
  ["demote"]="/demote #general @bugsbouncy @anotheruser #management-channel"
  ["add-to-all-org-channels"]="/add-to-all-org-channels @bugsbouncy"
)

# Generate Bruno file for each command
for cmd in "${!commands[@]}"; do
  example="${commands[$cmd]}"
  filename="$BRUNO_DIR/${cmd}.bru"

  cat > "$filename" << EOF
meta {
  name: process-slash-command-${cmd}
  type: http
  seq: 1
}

post {
  url: {{BASE_URL}}/slash-commands/process
  body: json
  auth: inherit
}

body:json {
  {
    "command": "${example}"
  }
}

settings {
  encodeUrl: true
  timeout: 0
}
EOF

  echo "Created: $filename"
done

echo ""
echo "✅ Generated ${#commands[@]} Bruno files in $BRUNO_DIR"
echo ""
echo "Files created:"
ls -la "$BRUNO_DIR"/*.bru 2>/dev/null || echo "No .bru files found"
