#!/bin/bash

# Zero-Touch Deployment Script for High-Performance News Website
# This script provides a convenient wrapper around the deployment agent

set -e

# Default values
CONFIG_FILE="deploy.yaml"
ACTION="deploy"
TARGET=""
VERSION=""
DRY_RUN=false
VERBOSE=false
LIMIT=10
FORMAT="text"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Zero-Touch Deployment Script for High-Performance News Website

OPTIONS:
    -c, --config FILE       Deployment configuration file (default: deploy.yaml)
    -a, --action ACTION     Action to perform: deploy, rollback, health-check, setup, status, list-deployments, validate (default: deploy)
    -t, --target TARGET     Target server name (required)
    -v, --version VERSION   Version to rollback to (required for rollback action)
    -d, --dry-run          Perform a dry run without making changes
    -l, --limit NUMBER      Limit for list operations (default: 10)
    -f, --format FORMAT     Output format: text, json (default: text)
    -V, --verbose          Enable verbose output
    -h, --help             Show this help message

EXAMPLES:
    # Deploy to production server
    $0 --target production

    # Deploy to staging with dry run
    $0 --target staging --dry-run

    # Rollback to previous version
    $0 --action rollback --target production --version news-website-1640995200

    # Check health of production server
    $0 --action health-check --target production

    # Setup a new server
    $0 --action setup --target new-server

    # Use custom config file
    $0 --config custom-deploy.yaml --target production

    # Get server status
    $0 --action status --target production

    # List recent deployments
    $0 --action list-deployments --target production --limit 5

    # Validate configuration
    $0 --action validate

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        -a|--action)
            ACTION="$2"
            shift 2
            ;;
        -t|--target)
            TARGET="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -l|--limit)
            LIMIT="$2"
            shift 2
            ;;
        -f|--format)
            FORMAT="$2"
            shift 2
            ;;
        -V|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate required arguments (except for validate action)
if [[ -z "$TARGET" && "$ACTION" != "validate" ]]; then
    log_error "Target server is required"
    usage
    exit 1
fi

if [[ "$ACTION" == "rollback" && -z "$VERSION" ]]; then
    log_error "Version is required for rollback action"
    usage
    exit 1
fi

# Check if config file exists
if [[ ! -f "$CONFIG_FILE" ]]; then
    log_error "Configuration file not found: $CONFIG_FILE"
    if [[ -f "deploy.yaml.example" ]]; then
        log_info "Example configuration file found. Copy it to $CONFIG_FILE and customize:"
        log_info "cp deploy.yaml.example $CONFIG_FILE"
    fi
    exit 1
fi

# Check if deployment binary exists
DEPLOY_BINARY="./cmd/deploy/deploy"
if [[ ! -f "$DEPLOY_BINARY" ]]; then
    log_info "Deployment binary not found. Building..."
    if ! go build -o "$DEPLOY_BINARY" ./cmd/deploy; then
        log_error "Failed to build deployment binary"
        exit 1
    fi
    log_success "Deployment binary built successfully"
fi

# Prepare deployment command
DEPLOY_CMD="$DEPLOY_BINARY --config $CONFIG_FILE --action $ACTION --target $TARGET"

if [[ "$DRY_RUN" == true ]]; then
    DEPLOY_CMD="$DEPLOY_CMD --dry-run"
fi

if [[ -n "$VERSION" ]]; then
    DEPLOY_CMD="$DEPLOY_CMD --version $VERSION"
fi

if [[ "$LIMIT" != "10" ]]; then
    DEPLOY_CMD="$DEPLOY_CMD --limit $LIMIT"
fi

if [[ "$FORMAT" != "text" ]]; then
    DEPLOY_CMD="$DEPLOY_CMD --format $FORMAT"
fi

# Show deployment information
log_info "Deployment Configuration:"
log_info "  Config File: $CONFIG_FILE"
log_info "  Action: $ACTION"
if [[ -n "$TARGET" ]]; then
    log_info "  Target: $TARGET"
fi
if [[ -n "$VERSION" ]]; then
    log_info "  Version: $VERSION"
fi
if [[ "$ACTION" == "list-deployments" ]]; then
    log_info "  Limit: $LIMIT"
fi
if [[ "$ACTION" == "status" || "$ACTION" == "list-deployments" ]]; then
    log_info "  Format: $FORMAT"
fi
log_info "  Dry Run: $DRY_RUN"

# Confirm deployment (unless it's a read-only action or dry run)
if [[ "$ACTION" != "health-check" && "$ACTION" != "status" && "$ACTION" != "list-deployments" && "$ACTION" != "validate" && "$DRY_RUN" != true ]]; then
    echo
    read -p "Do you want to proceed with the deployment? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Deployment cancelled"
        exit 0
    fi
fi

# Execute deployment
log_info "Starting deployment..."
echo

if [[ "$VERBOSE" == true ]]; then
    log_info "Executing: $DEPLOY_CMD"
fi

# Run the deployment command
if $DEPLOY_CMD; then
    echo
    case $ACTION in
        deploy)
            log_success "Deployment completed successfully!"
            ;;
        rollback)
            log_success "Rollback completed successfully!"
            ;;
        health-check)
            log_success "Health check passed!"
            ;;
        setup)
            log_success "Server setup completed successfully!"
            ;;
        status)
            log_success "Server status retrieved successfully!"
            ;;
        list-deployments)
            log_success "Deployment history retrieved successfully!"
            ;;
        validate)
            log_success "Configuration validation completed successfully!"
            ;;
    esac
else
    echo
    log_error "Deployment failed!"
    exit 1
fi

# Post-deployment actions
if [[ "$ACTION" == "deploy" && "$DRY_RUN" != true ]]; then
    echo
    log_info "Post-deployment recommendations:"
    log_info "  1. Monitor application logs: journalctl -u news-website -f"
    log_info "  2. Check application health: $0 --action health-check --target $TARGET"
    log_info "  3. Monitor system resources: htop, iostat, free -h"
    log_info "  4. Verify website functionality in browser"
fi