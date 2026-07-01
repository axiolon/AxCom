# Inventory Domain Models

This package defines the core domain models, interfaces, and rules for the inventory subdomain of the ecom-engine.

## Core Models

### StockItem

`StockItem` represents the stock level and backorder configuration for a specific product variant at a particular location.

- **Attributes**:
  - `VariantID`: Unique identifier for the product variant.
  - `LocationID`: Unique identifier for the physical location (warehouse/store).
  - `Quantity`: Current on-hand stock quantity.
  - `LowStockThreshold`: Threshold below which the item is flagged as low stock (typically default to 5 in the application layer).
  - `AllowBackorders`: Boolean flag allowing customers to purchase beyond available stock.
  - `BackorderLimit`: Maximum allowed backordered quantity.

- **Rules & Operations**:
  - `IsLowStock()`: Returns `true` if `Quantity` is at or below `LowStockThreshold`.
  - `Reserve(stock, qty)`: Attempts to deduct `qty` from stock. Validates that quantity is greater than zero and that stock is sufficient (accounting for backorder limits if enabled).
  - `Update(stock, qty)`: Updates the stock quantity with validation (must be non-negative).

### Reservation

`Reservation` represents a temporary lock on stock for a variant to prevent double-selling during checkout.

- **Attributes**:
  - `ID`: Unique reservation identifier.
  - `VariantID`: Unique identifier for the reserved product variant.
  - `LocationID`: Target warehouse/store location.
  - `Quantity`: The quantity being reserved.
  - `ExpiresAt`: Time when the reservation becomes invalid.

- **Rules & Operations**:
  - `ValidateReservation(r)`: Checks that the variant ID is not empty, the quantity is greater than zero, and the expiration time is in the future.

### Alert

`Alert` represents system notifications or events (such as low stock) that occur during inventory processing.

- **Attributes**:
  - `ID`: Unique identifier for the alert.
  - `Type`: Alert categorization (e.g., `"low_stock"`).
  - `Message`: Description of the alert.
  - `VariantID`: Product variant associated with the alert.
  - `CreatedAt`: Time the alert was triggered.
  - `IsRead`: Boolean read status of the alert.

- **Interfaces**:
  - `AlertDispatcher`: Interface defining the execution of alert delivery:
    ```go
    type AlertDispatcher interface {
        Dispatch(ctx context.Context, alert Alert) error
    }
    ```

### StockHistory

`StockHistory` records individual stock change events for auditing and tracking.

- **Attributes**:
  - `ID`: Unique identifier for the historical record.
  - `VariantID`: Unique identifier for the variant.
  - `LocationID`: Unique location identifier.
  - `OldQuantity`: Quantity before the change.
  - `NewQuantity`: Quantity after the change.
  - `ChangeReason`: Description of why the stock changed.
  - `ChangedBy`: Actor or system ID that made the change.
  - `ChangedAt`: Time the change was recorded.

- **Rules & Operations**:
  - `ValidateHistory(h)`: Verifies that `ID`, `VariantID`, `LocationID`, and `ChangeReason` are present.
