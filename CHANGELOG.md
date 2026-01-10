# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Docker output now streams in real-time during deployments instead of being buffered
- Users see image pull progress, container creation, and warnings as they happen
- No configuration needed - Docker automatically formats output for your environment

### Fixed
- Deployments with large images no longer appear frozen during pull operations
- Docker warnings and deprecation notices are now visible during deployment
- Error messages are shown immediately as they occur instead of only at the end
