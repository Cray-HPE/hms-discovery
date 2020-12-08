# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.8] - 2020-12-08
### Changed
- CASMINST-568: Update hms-dns-dhcp library version to pull in fix for patching Ethernet Interfaces that have MAC addresses with colons.

## [1.2.7] - 2020-12-4
### Changes
- Pin image dtr.dev.cray.com/cray/hms-mountain-discovery to 0.2.0 for v1.4 release.

## [1.2.6] - 2020-11-18
### Changed
- CASMINST-332: Fail gracefully when unable to communicate with a management switch. If a communication problem occurs with a management switch it will be ignored, and the discovery job will continue to try to communicate with the other switches. 
- Removed workaround for CASMHMS-3617, we no longer need to delete ethernet interface entries in SMD when updating component IDs.

## [1.2.5] - 2020-10-20

### Security
- CASMHMS-4105 - Updated base Golang Alpine image to resolve libcrypto vulnerability.

## [1.2.4] - 2020-09-04

### Security
- CASMHMS-2991 - Updated hms-discovery to use latest trusted baseOS images.

## [1.2.3] - 2020-08-20

### Added
- CASMHMS-3825: Added support for discovering Server Tech PDUs via RTS

### Added

### Changed

### Deprecated

### Removed

### Fixed

### Security