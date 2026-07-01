# Product Reviews Feature Tests

This document tracks and describes the testing strategy and validation suite for the `reviews` catalog feature.

---

## Overview

The `reviews` catalog feature allows customers to leave ratings and comments on products, view rating summaries, and allows merchants to reply to reviews.
- **Unit Tests (`service_test.go`)**: Validates validation constraints on ratings (1-5 range), non-empty comments, product existence verification, deletion permissions (owner, non-owner, admin overrides), and reply creation.
- **Integration Tests (`controller_test.go`)**: Validates routing, HTTP handlers, path parameters, request bodies, authentication/authorization checks, and E2E review flows.

---

## Reviews Feature Test Suites

### 1. Service Layer Unit Tests (`service_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **CTL-REV-SRV-ADD-001** | Submit review successfully | Product ID, User ID, rating: 5, comment: "Outstanding product!" | Review created successfully | Positive |
| **CTL-REV-SRV-ADD-002** | Submit review fails - invalid rating | Rating: 0 or outside 1-5 range | Error: rating must be between 1 and 5 | Negative |
| **CTL-REV-SRV-ADD-003** | Submit review fails - empty comment | Rating: 4, comment: "" | Error: comment is required | Negative |
| **CTL-REV-SRV-ADD-004** | Submit review fails - product not found | Non-existent product ID | Error: product not found | Negative |
| **CTL-REV-SRV-AVG-001** | Calculate average rating | Multiple reviews submitted | Verifies count and average rating calculations are correct | Positive |
| **CTL-REV-SRV-DEL-001** | Owner deletes review successfully | Valid review ID, correct Owner User ID | Review deleted | Positive |
| **CTL-REV-SRV-DEL-002** | Non-owner delete fails | Valid review ID, incorrect Owner User ID | Error: unauthorized delete | Negative |
| **CTL-REV-SRV-DEL-003** | Admin deletes review successfully | Valid review ID, admin override = true | Review deleted by admin | Positive |
| **CTL-REV-SRV-RPL-001** | Submit reply successfully | Review ID, reply body (User ID, comment) | Reply attached to the review in DB | Positive |
| **CTL-REV-SRV-RPL-002** | Submit reply fails - empty comment | Review ID, reply comment: "" | Error: reply comment is required | Negative |

### 2. HTTP Handler Integration & E2E Tests (`controller_test.go`)

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **CTL-REV-API-ADD-001** | Submit review endpoint success | POST `/api/products/:id/reviews` with valid JSON | HTTP 200, created review | Positive |
| **CTL-REV-API-GET-001** | Get reviews endpoint success | GET `/api/products/:id/reviews` | HTTP 200, array of reviews | Positive |
| **CTL-REV-API-SUM-001** | Get rating summary success | GET `/api/products/:id/reviews/summary` | HTTP 200, rating count and average score | Positive |
| **CTL-REV-API-DEL-001** | Delete review endpoint success | DELETE `/api/reviews/:id` | HTTP 200, success message | Positive |
| **CTL-REV-API-RPL-001** | Submit reply endpoint success | POST `/api/reviews/:id/reply` with valid JSON | HTTP 200, updated review with reply | Positive |
| **CTL-REV-E2E-FLOW-001** | Full review-reply-delete E2E flow | Multi-step request sequence: POST review -> GET check -> POST reply -> DELETE review | Verifies full state transition and cascading integrity | Positive |
