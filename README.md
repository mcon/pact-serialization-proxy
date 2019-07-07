# pact-serialization-proxy
[![Build Status](https://travis-ci.org/mcon/pact-serialization-proxy.svg?branch=master)](https://travis-ci.org/mcon/pact-serialization-proxy)

## Motivation

The Pact testing framework was designed with JSON over HTTP in mind - anyone using non-JSON encodings, and transports other than HTTP are unable to use Pact (with the exception of *message pacts* which are only suitable for async communication e.g. "put this thing on a queue for me" - even these *message pacts* often require use of JSON too).

It's desirable to be able to write Pact tests for Protobuf over HTTP or Protobuf over GRPC: just having a "contract" between services, in the form of `.proto` files isn't sufficient - this contract must be made explicit and verified. Typically testing contracts between services requires either integration tests, or "contract tests", as Pact facilitates - it would be nice to be able to do contract tests

## Aims
### Short term

To enable testing of Protobuf over HTTP with Pact - for various reasons, that's the setup I'd actually like to test.

### Longer term

Enable testing of Protobuf over GRPC: exploratory work has been done which leads me to believe this is possible without too much effort, however as I don't have a direct need for it, contributions are encouraged for this element.

## Design

In order to re-use as much of the Pact testing infrastructure as possible, a single new component will be added (pact-serialization-proxy) which will convert between Protobuf and JSON.

More details can be found [here](https://docs.google.com/presentation/d/13rTmXp7Gdcd_hHC_0YP5FR0fB1KIjZf9p5yrucjTlSc/edit#slide=id.g52499222dc_0_531).

"High level design diagram":
![diagram](https://github.com/mcon/pact-serialization-proxy/blob/master/architecture-diagram.png)

## Status

Currently still a work-in-progress, the following functionality is working with the C# Pact library (more details to come):
- Create a protobuf-based pact for GET requests.
- Verify protobuf-based pacts for GET requests.

The following work is outstanding:
- v0.1 release:
  - Add support for requests which contain non-empty request bodies (POST/PUT).
  - Add logging which allows failures to be debugged more easily.
- v0.2 release:
  - Modularize the code and add unit tests.
  - Ensure failure-cases are tested and behave as expected.
  - Add release deployment code to Travis-CI.
- v1.0 release:
  - Add an integration test between the serialization proxy and the Ruby Core.
  - Add documentation for:
    - How to add language support for the serialization proxy.
    - If accepted, then in the Pact spec and on the Pact website.
- Future release:
  - Support for GRPC.