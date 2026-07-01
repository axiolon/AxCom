# User Orders HTTP Controller Tests Matrix

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **ORD-USR-API-CRT-001** | Successful order checkout | Valid JSON payload with items, authenticated customer ID in context | Status 200 OK, returns JSON response with `success: true` and mapped OrderResponse | Positive |
| **ORD-USR-API-CRT-002** | Checkout fails - unauthorized | Valid JSON payload, but missing customer ID in context | Status 401 Unauthorized, returns JSON response with `success: false` | Negative |
| **ORD-USR-API-CRT-003** | Checkout fails - invalid request payload | Malformed JSON or fields that violate validation rules (e.g. negative price) | Status 400 Bad Request, returns JSON response with `success: false` | Negative |
| **ORD-USR-API-CRT-004** | Checkout fails - service logic error | Valid payload and customer, but Service returns error (e.g. empty order validation) | Status 400 Bad Request (or 500 depending on error type), returns JSON response with `success: false` | Negative |
| **ORD-USR-API-GET-001** | Get order details success | Existing order ID, authenticated customer ID matching order owner | Status 200 OK, returns JSON response with `success: true` and mapped OrderResponse | Positive |
| **ORD-USR-API-GET-002** | Get order fails - unauthorized | Existing order ID, missing customer ID in context | Status 401 Unauthorized, returns JSON response with `success: false` | Negative |
| **ORD-USR-API-GET-003** | Get order fails - forbidden ownership | Existing order ID, authenticated customer ID that does NOT match the order owner | Status 403 Forbidden, returns JSON response with `success: false` and "you do not have access to this order" | Negative |
| **ORD-USR-API-GET-004** | Get order fails - not found | Non-existent order ID | Status 404 Not Found, returns JSON response with `success: false` | Negative |
| **ORD-USR-API-LST-001** | List customer orders success | Authenticated customer ID in context | Status 200 OK, returns list of customer orders with `success: true` and count | Positive |
| **ORD-USR-API-LST-002** | List customer orders fails - unauthorized | Missing customer ID in context | Status 401 Unauthorized, returns JSON response with `success: false` | Negative |
