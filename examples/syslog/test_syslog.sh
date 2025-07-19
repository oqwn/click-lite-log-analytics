#!/bin/bash

# Test syslog ingestion using system logger command
# This script demonstrates sending logs via syslog protocol

SYSLOG_SERVER="localhost"
SYSLOG_PORT="20004"

echo "Sending test logs to syslog server at $SYSLOG_SERVER:$SYSLOG_PORT"

# Using logger command (if available)
if command -v logger &> /dev/null; then
    echo "Using system logger command..."
    
    # Send various priority logs
    logger -n $SYSLOG_SERVER -P $SYSLOG_PORT -p local0.info "Test message from shell script - INFO level"
    logger -n $SYSLOG_SERVER -P $SYSLOG_PORT -p local0.warning "Test message from shell script - WARNING level"
    logger -n $SYSLOG_SERVER -P $SYSLOG_PORT -p local0.err "Test message from shell script - ERROR level"
    logger -n $SYSLOG_SERVER -P $SYSLOG_PORT -p local0.debug "Test message from shell script - DEBUG level"
    
    # Send with different facilities
    logger -n $SYSLOG_SERVER -P $SYSLOG_PORT -p user.info "User facility message"
    logger -n $SYSLOG_SERVER -P $SYSLOG_PORT -p daemon.notice "Daemon facility message"
    logger -n $SYSLOG_SERVER -P $SYSLOG_PORT -p auth.warning "Auth facility warning"
    
    echo "Sent test messages using logger command"
else
    echo "logger command not found, using netcat instead..."
fi

# Using netcat for more control
if command -v nc &> /dev/null; then
    echo "Sending custom syslog messages using netcat..."
    
    # Function to send syslog message
    send_syslog() {
        local priority=$1
        local message=$2
        local timestamp=$(date '+%b %d %H:%M:%S')
        local hostname=$(hostname)
        
        # RFC3164 format
        echo "<$priority>$timestamp $hostname test-script[$$]: $message" | nc -u -w1 $SYSLOG_SERVER $SYSLOG_PORT
    }
    
    # Send various messages
    send_syslog 134 "Custom syslog message via netcat - INFO"  # 134 = local0.info
    send_syslog 132 "Custom syslog message via netcat - WARNING"  # 132 = local0.warning
    send_syslog 131 "Custom syslog message via netcat - ERROR"  # 131 = local0.err
    
    # Simulate application logs
    echo "Simulating application logs..."
    send_syslog 134 "Application starting up"
    sleep 1
    send_syslog 134 "Connected to database successfully"
    sleep 1
    send_syslog 134 "Server listening on port 8080"
    sleep 1
    send_syslog 132 "High memory usage detected: 85%"
    sleep 1
    send_syslog 134 "Request processed successfully"
    
    echo "Sent custom syslog messages"
else
    echo "Neither logger nor nc (netcat) found. Please install one of them."
    exit 1
fi

echo "Syslog test completed!"