# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.18.0] - 2025-03-13

### Security

- Updated image and module dependencies for security updates
- Various code changes to accomodate module updates
- Resolved build warnings in Dockerfiles and docker compose files
- Fixed docker-compose.devel.yaml file so that local dev testing works again
- Updated local dev testing readme for accuracy

## [1.17.0] - 2024-12-03

### Changed

- Updated go to 1.23

## [1.16.0] - 2024-02-07

### Changed
- Changed to not write the default credentials when there are already credentials in vault.

## [1.15.0] - 2023-09-25

### Added
- Added Vault loader (copied over from REDS) to populate default River Redfish and SNMP credentials into vault.
- Discover Management Nodes just like REDS
- Discover Management Virtual Nodes
- Populate per-switch credentials in Vault like REDS.

### Changed
- Updated hms-sls to v2 Go module.
- Migrated to hms-xname and hms-base/v2

## [1.14.0] - 2022-08-01

### Changed

- Updated hms-mountain-discovery image to v0.7.0

## [1.13.0] - 2022-07-28

### Changed
- CASMHMS-5613: Remove the hardcode of the HSM, SLS, and CAPMC endpoints for the mountain discovery job

## [1.12.0] - 2022-07-06

### Changed

- Changed HSM v1 API references to v2

## [1.11.0] - 2022-05-04

### Changed

- Updated hms-discovery to build using GitHub Actions instead of Jenkins.
- Pull images from artifactory.algol60.net instead of arti.dev.cray.com.

## [1.10.0] - 2021-11-19

### Added

- Support for HPE PDUs

## [1.9.0] - 2021-11-03

### Fixed

- Pick up new mountain discovery script that ignores disabled components

## [1.8.0] - 2021-10-29

### Changed

- bumped semver to adhere to semver standard

## [1.7.7] - 2021-09-08

### Changed

- Changed the docker image to run as the user nobody

## [1.7.6] - 2021-08-11

### Changed

- Added helm add repo to Makefile

## [1.7.5] - 2021-08-09

### Changed

- Added GitHub configuration files and fixed snyk warning.

## [1.7.4] - 2021-07-27

### Changed

- Github transition phase 3. Remove stash references.

## [1.7.3] - 2021-07-20

### Changed

- Add support for building within the CSM Jenkins.

## [1.7.2] - 2021-07-12

### Security
- CASMHMS-4933 - Updated base container images for security updates.

## [1.7.1] - 2021-06-07

### Changed
- CASMHMS-4907 - The discovery job will now check to see if an unknown ethernet interface has Redfish before updating the EthernetInterface in HSM to give its identify. This will prevent a situation where the EthernetInterface in HSM is given a component ID, but a Redfish Endpoint was not created for it.

## [1.7.0] - 2021-06-07

### Changed
- Bump minor version for CSM 1.1 release branch

## [1.6.0] - 2021-06-07

### Changed
- Bump minor version for CSM 1.1 release branch

## [1.5.4] - 2021-05-17

### Changed
- CASMHMS-4472 - Check for unknownComponents to respond to redfish before adding them to HSM.

## [1.5.3] - 2021-05-04

### Changed
- Updated docker-compose files to pull images from Artifactory instead of DTR.

## [1.5.2] - 2021-04-16

### Changed
- Updated Dockerfile to pull base images from Artifactory instead of DTR.

## [1.5.1] - 2021-04-13

### Security
- CASMHMS-4714 - Bump version to rebuild service and pull in security update.

## [1.5.0] - 2021-02-04

### Changed
- Update Copyright/License in source files
- Re-vendor go packages

## [1.4.0] - 2021-01-14

### Changed
- Updated license file.

## [1.3.0] - 2020-12-08

### Changed
- CASMINST-568: Update hms-dns-dhcp library version to pull in fix for patching Ethernet Interfaces that have MAC addresses with colons.

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
