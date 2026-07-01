# Orders Service Unit Tests Matrix

| ID | Scenario | Input | Expected Result | Type |
|---|---|---|---|---|
| **ORD-SRV-CRT-001** | Successful creation of a customer order | Customer ID and non-empty valid item array | Persists order, calculates correct total, publishes `order.created` event, returns order object, nil error | Positive |
| **ORD-SRV-CRT-002** | Validation failure - empty item list | Empty items array | Returns ErrEmptyOrder, bad request error, no DB persist, no event published | Negative |
| **ORD-SRV-CRT-003** | Validation failure - missing variant ID | Items containing blank variant ID | Returns ErrVariantIDRequired, bad request error | Negative |
| **ORD-SRV-CRT-004** | Validation failure - non-positive quantity | Items containing zero or negative quantity | Returns ErrInvalidQuantity, bad request error | Negative |
| **ORD-SRV-CRT-005** | Validation failure - negative price | Items containing negative price | Returns ErrInvalidPrice, bad request error | Negative |
| **ORD-SRV-CRT-006** | Database connection failure | Valid items, but DB insert fails | Returns database/internal error, no event published | Negative |
| **ORD-SRV-TRN-001** | Successful order state transition | Existing order ID, valid transition action (e.g. "pay") | Transitions order status to "paid", updates database, publishes `order.paid` event, returns order, nil error | Positive |
| **ORD-SRV-TRN-002** | Transition fails - order not found | Non-existent order ID, transition action | Returns ErrOrderNotFound, 404 error | Negative |
| **ORD-SRV-TRN-003** | Transition fails - invalid action | Existing order ID, invalid transition action (e.g. "ship" on pending) | Returns ErrInvalidTransition, bad request error | Negative |
| **ORD-SRV-TRN-004** | Transition fails - database update error | Existing order, valid transition, but DB update fails | Returns database/internal error, no event published | Negative |
| **ORD-SRV-GET-001** | Retrieve order success | Existing order ID | Returns order, nil error | Positive |
| **ORD-SRV-GET-002** | Retrieve order not found | Non-existent order ID | Returns ErrOrderNotFound, nil order | Negative |
| **ORD-SRV-LST-001** | List customer orders success | Existing customer ID | Returns slice of customer orders, nil error | Positive |
| **ORD-SRV-LST-002** | List customer orders DB failure | Existing customer ID, DB error | Returns database/internal error | Negative |
| **ORD-SRV-LST-003** | List all orders globally success | None | Returns slice of all orders globally, nil error | Positive |
| **ORD-SRV-LST-004** | List all orders globally DB failure | None, DB error | Returns database/internal error | Negative |
| **ORD-SRV-EVT-001** | Event subscription triggers order status update | Payment succeeded event payload | Calls TransitionOrder to transition order to "paid", updates database, publishes `order.paid` | Positive |
