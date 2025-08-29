#!/bin/bash

# LocalStack SQS Setup Script
# This script runs when LocalStack is ready

set -e

echo "########### Setting up SQS Queues ###########"

# Configuration
QUEUE_NAME="weather-queue"

echo "Creating SQS queue: $QUEUE_NAME"

# Create the SQS queue (simplified - no custom attributes for now)
awslocal sqs create-queue --queue-name="$QUEUE_NAME"

echo "✅ Queue '$QUEUE_NAME' created successfully"

# List all queues to verify
echo "########### Current SQS Queues ###########"
awslocal sqs list-queues

echo "########### SQS setup completed! ###########"
