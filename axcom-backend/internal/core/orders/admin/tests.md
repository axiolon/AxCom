# Admin Orders HTTP Controller Tests Matrix

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **ORD-ADM-API-LST-001** | List all orders success | None | Status 200 OK, returns JSON response with `success: true`, list of all orders, and count | Positive |
| **ORD-ADM-API-LST-002** | List all orders resolves guest details | None, orders include a guest order | Status 200 OK, guest order has resolved guest customer info in output | Positive |
| **ORD-ADM-API-GET-001** | Get specific order success | Existing order ID | Status 200 OK, returns JSON response with `success: true` and mapped OrderResponse | Positive |
| **ORD-ADM-API-GET-002** | Get order fails - missing order ID parameter | Empty order ID in request route | Status 400 Bad Request, returns JSON response with `success: false` | Negative |
| **ORD-ADM-API-GET-003** | Get order fails - order not found | Non-existent order ID | Status 404 Not Found, returns JSON response with `success: false` | Negative |
| **ORD-ADM-API-TRN-001** | Successful order state transition | Existing order ID, valid TransitionRequest action | Status 200 OK, returns transitioned OrderResponse, status: true | Positive |
| **ORD-ADM-API-TRN-002** | Transition fails - invalid request payload | Malformed JSON | Status 400 Bad Request | Negative |
| **ORD-ADM-API-TRN-003** | Transition fails - service/domain state machine rule violation | Valid transition request, but transition action is illegal in current state | Status 400 Bad Request (or error code returned by service) | Negative |
