# Product Images Feature Tests

This document tracks and describes the testing strategy and validation suite for the `images` catalog feature.

---

## Overview

The `images` catalog feature manages the lifecycle of product media, including requesting presigned file upload URLs, registering completed uploads into the database, setting a primary image, and deleting images.
- **Unit Tests (`service_test.go`)**: Validates validation rules, interaction with storage engine mock, auto-promotion of primary images on deletion, and repository updates.
- **Integration Tests (`controller_test.go`)**: Validates controller endpoints routing, request validation, S3/storage mock responses, and end-to-end media upload flow scenarios.

---

## Images Feature Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **CTL-IMG-SRV-PSG-001** | Presign upload URL success | Product ID, slice of `PresignImageRequest` (filename, content type) | Array of presigned URLs, public locations, and HTTP methods | Positive |
| **CTL-IMG-SRV-PSG-002** | Presign upload URL fails - empty request | Product ID, `nil` or empty slice of requests | Error 400: empty files/missing arguments | Negative |
| **CTL-IMG-SRV-REG-001** | Register uploaded images success | Product ID, array of `RegisterImageRequest` (key, isPrimary) | DB updated, public URLs generated, image models returned | Positive |
| **CTL-IMG-SRV-DEL-001** | Delete image successfully | Product ID, image ID to remove | Image removed from DB and file storage; remaining image promoted to primary if primary was deleted | Positive |
| **CTL-IMG-SRV-PRI-001** | Set primary image successfully | Product ID, image ID to promote | Target image set to primary (`IsPrimary = true`), others unset to `false` in repository | Positive |

### 2. HTTP Handler Integration & E2E Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **CTL-IMG-API-PSG-001** | Presign upload endpoint success | POST `/api/products/:id/images/presign` with valid payload | HTTP 200, array of presigned upload URLs | Positive |
| **CTL-IMG-API-REG-001** | Register images endpoint success | POST `/api/products/:id/images/register` with valid payload | HTTP 200, array of registered image meta-objects | Positive |
| **CTL-IMG-E2E-FLOW-001** | E2E Image upload and delete flow | Sequence: POST presign -> POST register -> DELETE image | Full workflow validation against mock storage and DB | Positive |
