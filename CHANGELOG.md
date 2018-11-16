# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).
This `CHANGELOG.md` implements the spirit of http://keepachangelog.com/.

## [1.23](https://github.com/Comcast/eel/compare/v1.22...dev) - [Unreleased]

### Updated
* reduce log lines

## [1.22](https://github.com/Comcast/eel/compare/v1.21...v1.22) - 2018-11-12

### Added
* Add function base64decode
* Add function loadFile
* Add function hmac

## [1.21](https://github.com/Comcast/eel/compare/v1.20...v1.21) - 2018-07-09

## [1.20](https://github.com/Comcast/eel/compare/v1.19...v1.20) - 2018-06-18

## [1.19](https://github.com/Comcast/eel/compare/v1.18...v1.19) - 2018-05-24

## [1.18](https://github.com/Comcast/eel/compare/v1.17...v1.18) - 2018-05-07

## [1.17](https://github.com/Comcast/eel/compare/v1.16...v1.17) - 2018-03-26

### Added
* Gears suppport

## [1.16](https://github.com/Comcast/eel/compare/v1.15...v1.16) - 2018-03-05

### Added
* Added a new config parameter: ElementsAuth

### Fixed
* XRULES-10493: support escape character in function parameters

## [1.15](https://github.com/Comcast/eel/compare/v1.14...v1.15) - 2017-12-01

### Fixed
* Update retry to not retry for 300 and 400

## [1.14](https://github.com/Comcast/eel/compare/v1.13...v1.14) - 2017-11-08

### Fixed
* Fixed bad urls in `CHANGELOG.md`

### Changed
* Commented out array_length_error log to address XRULES-9749

## [1.13](https://github.com/Comcast/eel/compare/v1.12...v1.13) - [2017-09-08]

### Changed
* add tenantId in logs

## [1.12](https://github.com/Comcast/eel/compare/v1.11...v1.12) - [2017-07-28]

### Fixed
* XRULES-8923: erroneous metric on published events

## [1.11](https://github.com/Comcast/eel/compare/v1.10...v1.11) - [2017-06-19]

### Added
* This `CHANGELOG.md` file
* .github/PULL_REQUEST_TEMPLATE.md to improve our PR process

### Fixed
* XRULES-8388: Panic handling code caused the code to panic
