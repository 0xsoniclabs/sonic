#!/bin/bash

# Configuration
TEST_COMMAND="go test ./... -count 1 -p 4 -timeout 0"
# The 'go test' command builds test binaries with random names like 'main.test', 'package.test', or similar.
# In Go modules, the main test binary is usually named 'main.test' or based on the package name.
# We will watch for any process containing 'test' and the Go environment's temporary directory.
PROCESS_NAME_PATTERN="test.count"
INTERVAL=0.001 # Monitor check interval in seconds

echo "--------------------------------------------------------"
echo "Go Test Parallelism Monitor"
echo "--------------------------------------------------------"
echo "Running command: $TEST_COMMAND"
echo "Monitoring interval: ${INTERVAL}s"
echo "--------------------------------------------------------"

# 1. Run the go test command in the background
# We must build the test binaries first so we know what process names to look for.
# Using 'go test -c' compiles the binaries without running them.
# However, the easiest way to ensure all dependencies and naming are correct is to just run the full command.

# The command is run with '&' to put it in the background
$TEST_COMMAND &
TEST_PID=$!
echo "Go Test command started with PID: $TEST_PID"

# 2. Monitoring Loop
echo -e "\n--- Live Parallelism Monitoring (Max Expected: 4) ---"
echo "Time | Total Test Processes Running | Max Parallel Seen"

LAST=0
HIGHEST=0

while kill -0 $TEST_PID 2>/dev/null; do
    # Get the number of running test binaries.
    # We look for processes named like the temporary test binary pattern 
    # and exclude the main 'go' command wrapper itself.
    
    # We use 'pgrep' which is cleaner than 'ps aux | grep'
    # We look for any running executable ending in '.test'
    CURRENT_PARALLEL=$(pgrep -f "${PROCESS_NAME_PATTERN}" | wc -l)

    # Note: On some systems, pgrep might match the current script or other unintended processes. 
    # For a more robust solution, one would monitor the specific package directories inside the temporary $GOCACHE, 
    # but this simple process name check works well in standard environments.

    if (( CURRENT_PARALLEL != LAST )); then
        LAST=$CURRENT_PARALLEL
        # Print status
        echo -e "$(date +%H:%M:%S) | \t\t${CURRENT_PARALLEL} \t\t  | \t  ${HIGHEST}"
        if (( CURRENT_PARALLEL > HIGHEST )); then
            HIGHEST=$CURRENT_PARALLEL
        fi
    fi


    sleep $INTERVAL
done

# 3. Final cleanup and report
echo -e "\n--- Test Command Finished ---"

# Wait for the main 'go test' process to fully exit and capture its exit code
wait $TEST_PID
TEST_EXIT_CODE=$?

echo "Final Go Test Exit Code: $TEST_EXIT_CODE"
echo "Maximum Parallel Packages Detected: $HIGHEST"
echo "--------------------------------------------------------"

exit $TEST_EXIT_CODE
