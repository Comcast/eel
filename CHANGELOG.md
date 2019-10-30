# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).
This `CHANGELOG.md` implements the spirit of http://keepachangelog.com/.

## [1.35](https://github.com/Comcast/eel/compare/v1.34.0...dev) - [Unreleased]

### Added
* XRULES-15127: Updates to Profile Service Endpoint

## [1.34](https://github.com/Comcast/eel/compare/v1.33.0...v1.34.0) - 2019-10-17

### Added
* Ability to specify a basepath in the URL

## [1.33](https://github.com/Comcast/eel/compare/v1.32.0...v1.33.0) - 2019-09-23

## [1.32](https://github.com/Comcast/eel/compare/v1.31.0...v1.32.0) - 2019-08-09

## [1.31](https://github.com/Comcast/eel/compare/v1.30.0...v1.31.0) - 2019-07-19

### Added
* Ability to send abitrary parameters to a pubblisher
* Remove kafka/asyncReply related parameters

## [1.30](https://github.com/Comcast/eel/compare/v1.29...v1.30.0) - 2019-06-07
* XRULES-13840: Default handlers in EEL
* XRULES-13983: Flow execution log issues

## [1.29](https://github.com/Comcast/eel/compare/v1.28...v1.29) - 2019-05-17

## [1.28](https://github.com/Comcast/eel/compare/v1.27...v1.28) - 2019-04-29

## [1.27](https://github.com/Comcast/eel/compare/v1.26...v1.27) - 2019-04-08

## [1.26](https://github.com/Comcast/eel/compare/v1.25...v1.26) - 2019-03-18

## [1.25](https://github.com/Comcast/eel/compare/v1.24...v1.25) - 2019-02-15

### Added
* Two new EEL functions: propExists and toTS

## [1.24](https://github.com/Comcast/eel/compare/v1.23...dev) - 2019-01-28

## Updated
* XRULES-12619: All components need to log Gears Portal application id

## [1.23](https://github.com/Comcast/eel/compare/v1.22...v1.23) - 2019-01-07

### Updated
* reduce log lines

### Added
* Add a global flag to disable all plugins except webhook

## [1.22](https://github.com/Comcast/eel/compare/v1.21...v1.22) - 2018-11-12

### Added
* Add function base64decode
* Add function loadFile
* Add function hmac
* Add optional multiple partitions for kafka publisher

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
