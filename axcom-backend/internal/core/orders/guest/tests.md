# Guest Orders HTTP Controller Tests Matrix

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **ORD-GST-API-CRT-001** | Successful guest checkout | Valid JSON payload with items and guest info | Status 200 OK, returns GuestOrderResponse with `success: true` containing order total, ID, and guest info | Positive |
| **ORD-GST-API-CRT-002** | Guest checkout fails - invalid request payload | Malformed JSON request body | Status 400 Bad Request, returns JSON response with `success: false` | Negative |
| **ORD-GST-API-CRT-003** | Guest checkout fails - validation error (missing name) | Guest info missing Name | Status 400 Bad Request, returns JSON response with `success: false` and message "guest name is required" | Negative |
| **ORD-GST-API-CRT-004** | Guest checkout fails - validation error (missing email) | Guest info missing Email | Status 400 Bad Request, returns JSON response with `success: false` and message "guest email is required" | Negative |
| **ORD-GST-API-CRT-005** | Guest checkout fails - validation error (missing contact number) | Guest info missing ContactNumber | Status 400 Bad Request, returns JSON response with `success: false` and message "guest contact number is required" | Negative |
| **ORD-GST-API-CRT-006** | Guest checkout fails - service creation error | Valid guest details, but Service returns error | Status 400 Bad Request (or 500), returns JSON response with `success: false` | Negative |
| **ORD-GST-API-CRT-007** | Guest checkout fails - guest repo persistence error | Order created, but guestRepo.Save fails | Status 500 Internal Server Error, returns JSON response with `success: false` and message "failed to save guest checkout info" | Negative |
