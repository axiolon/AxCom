# Axiolon Labs E-Commerce Engine - API Contract

This document provides a comprehensive API contract for the E-Commerce Engine services. It serves as the primary reference for frontend developers to integrate with the backend endpoints.

---

## 1. General Protocol & Response Envelopes

### Base URL
All API requests are prefixed with `/api`.

### Headers
- **Content-Type**: `application/json` (required for all `POST`/`PUT` requests)
- **Authorization**: `Bearer <JWT_ACCESS_TOKEN>` (required for all authenticated endpoints)

### Global Envelope Structure
Every response from the server follows a standard JSON wrapper:

#### Success Response Envelope
```json
{
  "success": true,
  "data": <DataPayload> // Object, Array, or primitive depending on the endpoint
}
```

#### Error Response Envelope
```json
{
  "success": false,
  "error": "User-friendly description of the error."
}
```

### Common HTTP Status Codes
* **`200 OK`**: Request succeeded.
* **`400 Bad Request`**: Request payload invalid, missing required fields, or validation failed.
* **`401 Unauthorized`**: Authentication is missing, malformed, or the JWT is expired/invalid.
* **`403 Forbidden`**: Authentication is valid, but the user lacks permission to access the resource (e.g. non-admin accessing admin routes).
* **`404 Not Found`**: The requested entity (Product, Category, Order, Stock Item) does not exist.
* **`409 Conflict`**: Action conflicts with system state (e.g., registering an email that already exists, duplicate SKUs).
* **`500 Internal Server Error`**: Unexpected server-side failures.

---

## 2. Authentication Services (Public)

### Register User
* **Endpoint**: `POST /api/auth/register`
* **Functionality**: Registers a new user account.
* **Request Body**:
  ```json
  {
    "email": "user@example.com",
    "password": "Password123!",
    "role": "customer" // Optional. Defaults to "customer" if omitted
  }
  ```
* **Response `data` Shape**:
  ```json
  {
    "id": "60a89d70-c75c-4d33-911e-a4c3f58a36ef",
    "email": "user@example.com",
    "role": "customer",
    "created_at": "2026-06-06T11:25:24.000Z"
  }
  ```
* **Error Codes**:
  * `400 Bad Request`: `"Invalid request payload"`, `"invalid email address"`, `"password must be at least 8 characters and contain at least one letter and one number"`.
  * `409 Conflict`: `"email already exists"`.

### Login
* **Endpoint**: `POST /api/auth/login`
* **Functionality**: Authenticate credentials and get Access and Refresh tokens.
* **Request Body**:
  ```json
  {
    "email": "user@example.com",
    "password": "Password123!"
  }
  ```
* **Response `data` Shape**:
  ```json
  {
    "access_token": "eyJhbGciOi...",
    "refresh_token": "a1b2c3d4e5...",
    "expires_at": "2026-06-06T11:40:24.000Z"
  }
  ```
* **Error Codes**:
  * `400 Bad Request`: `"Invalid request payload"`.
  * `401 Unauthorized`: `"invalid credentials"`.

### Logout
* **Endpoint**: `POST /api/auth/logout`
* **Functionality**: Revoke the specified refresh token and invalidate the session.
* **Request Body**:
  ```json
  {
    "refresh_token": "a1b2c3d4e5..."
  }
  ```
* **Response `data` Shape**:
  ```json
  {
    "message": "logged out successfully"
  }
  ```
* **Error Codes**:
  * `400 Bad Request`: `"Invalid request payload"`.
  * `401 Unauthorized`: `"token has been revoked"` or `"token has expired"`.

### Refresh Session
* **Endpoint**: `POST /api/auth/refresh`
* **Functionality**: Issue a new access token and refresh token using an active refresh token.
* **Request Body**:
  ```json
  {
    "refresh_token": "a1b2c3d4e5..."
  }
  ```
* **Response `data` Shape**: Same as **Login** response `data`.
* **Error Codes**:
  * `400 Bad Request`: `"Invalid request payload"`.
  * `401 Unauthorized`: `"token has expired"` or `"token has been revoked"`.

### Request Password Reset
* **Endpoint**: `POST /api/auth/password-reset`
* **Functionality**: Start a password reset process. *(Note: For local/testing environments, this response returns the generated reset token directly in the body).*
* **Request Body**:
  ```json
  {
    "email": "user@example.com"
  }
  ```
* **Response `data` Shape**:
  ```json
  {
    "message": "Password reset token generated successfully. In production, this would be emailed.",
    "reset_token": "reset-token-value-here",
    "expires_at": "2026-06-06T12:25:24.000Z"
  }
  ```
* **Error Codes**:
  * `400 Bad Request`: `"invalid email address"`.
  * `404 Not Found`: `"user not found"`.

### Confirm Password Reset
* **Endpoint**: `POST /api/auth/password-reset/confirm`
* **Functionality**: Provide the reset token and complete the password change.
* **Request Body**:
  ```json
  {
    "token": "reset-token-value-here",
    "new_password": "NewStrongPassword123!"
  }
  ```
* **Response `data` Shape**:
  ```json
  {
    "message": "password has been reset successfully"
  }
  ```
* **Error Codes**:
  * `400 Bad Request`: `"password must be at least 8 characters..."`.
  * `401 Unauthorized`: `"invalid reset token"`, `"reset token already used"`, `"token has expired"`.

---

## 3. Catalog & Categories

### List Products (Public)
* **Endpoint**: `GET /api/products`
* **Query Parameters**:
  * `category_id` (string, optional): Filter by category ID.
  * `price_min` (number, optional): Minimum variant price filter.
  * `price_max` (number, optional): Maximum variant price filter.
  * `attributes` (string, optional): Comma-separated list of key-value attributes (e.g. `size:L,color:black`).
* **Response `data` Shape**: Array of Products
  ```json
  [
    {
      "id": "prod-101",
      "name": "Classic Crewneck Tee",
      "description": "High-quality organic cotton crewneck tee.",
      "category_id": "cat-202",
      "variants": [
        {
          "id": "var-501",
          "sku": "TEE-BLK-L",
          "name": "Classic Crewneck Tee - Black - L",
          "price": 24.99,
          "attributes": {
            "color": "black",
            "size": "L"
          }
        }
      ]
    }
  ]
  ```

### Get Product Details (Public)
* **Endpoint**: `GET /api/products/:id`
* **Path Parameters**:
  * `id` (string): The ID of the product.
* **Response `data` Shape**: Single Product object (identical to List Products item).
* **Error Codes**:
  * `404 Not Found`: `"product not found"`.

### Create Product (Protected)
* **Endpoint**: `POST /api/products`
* **Security**: Required Bearer token.
* **Request Body**:
  ```json
  {
    "name": "Classic Crewneck Tee",
    "description": "High-quality organic cotton crewneck tee.",
    "category_id": "cat-202",
    "variants": [
      {
        "sku": "TEE-BLK-L",
        "name": "Classic Crewneck Tee - Black - L",
        "price": 24.99,
        "attributes": {
          "color": "black",
          "size": "L"
        }
      }
    ]
  }
  ```
* **Response `data` Shape**: Newly created Product object (with system generated IDs).
* **Error Codes**:
  * `400 Bad Request`: `"product name is required"`, `"product category ID is required"`, `"product must have at least one variant"`, `"duplicate SKU found in variants"`, `"price cannot be negative"`.
  * `401 Unauthorized`.

### Update Product (Protected)
* **Endpoint**: `PUT /api/products/:id`
* **Security**: Required Bearer token.
* **Path Parameters**:
  * `id` (string): The ID of the product to update.
* **Request Body**: Same structure as **Create Product**.
* **Response `data` Shape**: Updated Product object.
* **Error Codes**:
  * `400 Bad Request`.
  * `401 Unauthorized`.
  * `404 Not Found`: `"product not found"`.

### Delete Product (Protected)
* **Endpoint**: `DELETE /api/products/:id`
* **Security**: Required Bearer token.
* **Path Parameters**:
  * `id` (string): The ID of the product to delete.
* **Response `data` Shape**:
  ```json
  {
    "message": "product deleted"
  }
  ```
* **Error Codes**:
  * `401 Unauthorized`.
  * `404 Not Found`: `"product not found"`.

### List Categories (Public)
* **Endpoint**: `GET /api/categories`
* **Response `data` Shape**: Array of Category objects
  ```json
  [
    {
      "id": "cat-202",
      "name": "Apparel",
      "slug": "apparel",
      "parent_id": null // Can be string category-id of parent
    }
  ]
  ```

### Create Category (Protected)
* **Endpoint**: `POST /api/categories`
* **Security**: Required Bearer token.
* **Request Body**:
  ```json
  {
    "name": "Apparel",
    "slug": "apparel", // Optional. If empty, generated from name
    "parent_id": null // Optional
  }
  ```
* **Response `data` Shape**: Newly created Category object.
* **Error Codes**:
  * `400 Bad Request`: `"category name is required"`.
  * `401 Unauthorized`.

### Update Category (Protected)
* **Endpoint**: `PUT /api/categories/:id`
* **Security**: Required Bearer token.
* **Path Parameters**:
  * `id` (string): Category ID to update.
* **Request Body**: Same structure as **Create Category**.
* **Response `data` Shape**: Updated Category object.
* **Error Codes**:
  * `401 Unauthorized`.
  * `404 Not Found`: `"category not found"`.

### Delete Category (Protected)
* **Endpoint**: `DELETE /api/categories/:id`
* **Security**: Required Bearer token.
* **Path Parameters**:
  * `id` (string): Category ID to delete.
* **Response `data` Shape**:
  ```json
  {
    "message": "category deleted"
  }
  ```
* **Error Codes**:
  * `401 Unauthorized`.
  * `404 Not Found`: `"category not found"`.

### Search Products (Public)
* **Endpoint**: `GET /api/products/search`
* **Functionality**: Search products catalog. Aliases list products.
* **Query Parameters**: Same query parameters as list products.
* **Response `data` Shape**: Array of Products.

### Get Category Details (Public)
* **Endpoint**: `GET /api/categories/:id`
* **Path Parameters**:
  * `id` (string): The ID of the category.
* **Response `data` Shape**: Single Category object.

### Presign Upload URL for Product Image (Protected)
* **Endpoint**: `POST /api/products/:id/images/presign`
* **Security**: Required Bearer token & Admin/Staff role.
* **Request Body**:
  ```json
  {
    "filename": "image.jpg",
    "content_type": "image/jpeg"
  }
  ```
* **Response `data` Shape**:
  ```json
  {
    "upload_url": "https://pub-accountid.r2.dev/products/image.jpg?presigned=put",
    "public_url": "https://pub-accountid.r2.dev/products/image.jpg",
    "method": "PUT"
  }
  ```

### Register Uploaded Image Metadata (Protected)
* **Endpoint**: `POST /api/products/:id/images/register`
* **Security**: Required Bearer token & Admin/Staff role.
* **Request Body**:
  ```json
  {
    "url": "https://pub-accountid.r2.dev/products/image.jpg",
    "filename": "image.jpg"
  }
  ```
* **Response `data` Shape**: Registered Image object.

### Delete Product Image (Protected)
* **Endpoint**: `DELETE /api/products/:id/images/:imageId`
* **Security**: Required Bearer token & Admin/Staff role.
* **Response `data` Shape**: Message indicating deletion.

### Set Primary Product Image (Protected)
* **Endpoint**: `PUT /api/products/:id/images/:imageId/primary`
* **Security**: Required Bearer token & Admin/Staff role.
* **Response `data` Shape**: Updated Product object.

### Get Product Variants (Public)
* **Endpoint**: `GET /api/products/:id/variants`
* **Response `data` Shape**: Array of Product Variants.

### Add Product Variant (Protected)
* **Endpoint**: `POST /api/products/:id/variants`
* **Security**: Required Bearer token & Admin/Staff role.
* **Request Body**:
  ```json
  {
    "sku": "TEE-BLK-M",
    "name": "Classic Crewneck Tee - Black - M",
    "price": 24.99,
    "attributes": {
      "color": "black",
      "size": "M"
    }
  }
  ```
* **Response `data` Shape**: Newly created Variant object.

### Update Product Variant (Protected)
* **Endpoint**: `PUT /api/products/:id/variants/:variantId`
* **Security**: Required Bearer token & Admin/Staff role.
* **Request Body**: Same as Add Product Variant.
* **Response `data` Shape**: Updated Variant object.

### Delete Product Variant (Protected)
* **Endpoint**: `DELETE /api/products/:id/variants/:variantId`
* **Security**: Required Bearer token & Admin/Staff role.
* **Response `data` Shape**: Message indicating variant deletion.

### Apply Product Discount (Protected)
* **Endpoint**: `POST /api/products/:id/discount`
* **Security**: Required Bearer token & Admin/Staff role.
* **Request Body**:
  ```json
  {
    "discount_price": 19.99
  }
  ```
* **Response `data` Shape**: Updated Product object.

### Remove Product Discount (Protected)
* **Endpoint**: `DELETE /api/products/:id/discount`
* **Security**: Required Bearer token & Admin/Staff role.
* **Response `data` Shape**: Updated Product object.

### Bulk Create Products (Protected)
* **Endpoint**: `POST /api/products/bulk`
* **Security**: Required Bearer token & Admin/Staff role.
* **Request Body**: Array of Create Product payloads.
* **Response `data` Shape**: Array of created Products.

### Bulk Update Products (Protected)
* **Endpoint**: `PUT /api/products/bulk`
* **Security**: Required Bearer token & Admin/Staff role.
* **Response `data` Shape**: Array of updated Products.

### Bulk Delete Products (Protected)
* **Endpoint**: `DELETE /api/products/bulk`
* **Security**: Required Bearer token & Admin/Staff role.
* **Request Body**: Array of product IDs.
* **Response `data` Shape**: Message indicating bulk deletion success.

---

## 4. Product Reviews

### List Product Reviews (Public)
* **Endpoint**: `GET /api/products/:id/reviews`
* **Path Parameters**:
  * `id` (string): Product ID.
* **Response `data` Shape**: Array of Reviews
  ```json
  [
    {
      "id": "rev-901",
      "product_id": "prod-101",
      "user_id": "user-uuid",
      "rating": 5,
      "comment": "Perfect fit, very soft material!",
      "created_at": "2026-06-06T11:25:24.000Z"
    }
  ]
  ```

### Get Product Rating Summary (Public)
* **Endpoint**: `GET /api/products/:id/reviews/rating`
* **Path Parameters**:
  * `id` (string): Product ID.
* **Response `data` Shape**:
  ```json
  {
    "product_id": "prod-101",
    "average_rating": 4.8,
    "review_count": 25
  }
  ```

### Submit Product Review (Protected)
* **Endpoint**: `POST /api/products/:id/reviews`
* **Security**: Required Bearer token.
* **Path Parameters**:
  * `id` (string): Product ID.
* **Request Body**:
  ```json
  {
    "rating": 5, // Integer (1 to 5)
    "comment": "This tee is incredible!"
  }
  ```
* **Response `data` Shape**:
  ```json
  {
    "id": "rev-902",
    "product_id": "prod-101",
    "user_id": "user-uuid",
    "rating": 5,
    "comment": "This tee is incredible!",
    "created_at": "2026-06-06T11:25:24.000Z"
  }
  ```
  * `401 Unauthorized`.

### Update Product Review (Protected)
* **Endpoint**: `PUT /api/products/:id/reviews/:reviewId`
* **Security**: Required Bearer token.
* **Request Body**:
  ```json
  {
    "rating": 4,
    "comment": "Nice tee, but slightly different color."
  }
  ```
* **Response `data` Shape**: Updated Review object.

### Reply to Product Review (Protected - Admin only)
* **Endpoint**: `POST /api/products/:id/reviews/:reviewId/reply`
* **Security**: Required Bearer token & Admin role.
* **Request Body**:
  ```json
  {
    "comment": "Thank you for your feedback!"
  }
  ```
* **Response `data` Shape**: Newly created Reply/Comment object.

### Delete Product Review (Protected)
* **Endpoint**: `DELETE /api/products/:id/reviews/:reviewId`
* **Security**: Required Bearer token.
* **Response `data` Shape**: Message indicating deletion.

---

## 5. Shopping Cart (Protected - Customer only)

### Get Cart
* **Endpoint**: `GET /api/cart`
* **Security**: Required Bearer token.
* **Response `data` Shape**:
  ```json
  {
    "customer_id": "60a89d70-c75c-4d33-911e-a4c3f58a36ef",
    "items": [
      {
        "variant_id": "var-501",
        "quantity": 2,
        "price": 24.99
      }
    ]
  }
  ```
* **Error Codes**:
  * `401 Unauthorized`.

### Add/Update Item in Cart
* **Endpoint**: `POST /api/cart`
* **Security**: Required Bearer token.
* **Request Body**:
  ```json
  {
    "variant_id": "var-501",
    "quantity": 2,
    "price": 24.99
  }
  ```
* **Response `data` Shape**: Updated Cart object (same structure as Get Cart).
* **Error Codes**:
  * `400 Bad Request`: `"variant ID is required"`, `"quantity must be greater than zero"`, `"price cannot be negative"`.
  * `401 Unauthorized`.

### Clear Cart
* **Endpoint**: `DELETE /api/cart`
* **Security**: Required Bearer token.
* **Response `data` Shape**:
  ```json
  {
    "message": "cart cleared"
  }
  ```
  * `401 Unauthorized`.

### Get Cart Item Count (Protected)
* **Endpoint**: `GET /api/cart/count`
* **Security**: Required Bearer token.
* **Response `data` Shape**:
  ```json
  {
    "count": 5
  }
  ```

### Update Cart Item Quantity/Price (Protected)
* **Endpoint**: `PUT /api/cart/items/:variantId`
* **Security**: Required Bearer token.
* **Request Body**:
  ```json
  {
    "quantity": 3
  }
  ```
* **Response `data` Shape**: Updated Cart object.

### Remove Item from Cart (Protected)
* **Endpoint**: `DELETE /api/cart/items/:variantId`
* **Security**: Required Bearer token.
* **Response `data` Shape**: Updated Cart object.

### Merge Guest and Authenticated Cart (Protected)
* **Endpoint**: `POST /api/cart/merge`
* **Security**: Required Bearer token.
* **Request Body**:
  ```json
  {
    "guest_customer_id": "temp-guest-uuid"
  }
  ```
* **Response `data` Shape**: Merged Cart object.

---

## 6. Checkout & Orders

### Guest Checkout (Public)
* **Endpoint**: `POST /api/orders/guest`
* **Functionality**: Places an order for an unauthenticated user session.
* **Request Body**:
  ```json
  {
    "guest_info": {
      "name": "Jane Doe",
      "email": "jane.doe@example.com",
      "contact_number": "+15551234567"
    },
    "items": [
      {
        "variant_id": "var-501",
        "quantity": 2,
        "price": 24.99
      }
    ]
  }
  ```
* **Response `data` Shape**:
  ```json
  {
    "order_id": "order-777",
    "status": "pending",
    "total": 49.98,
    "created_at": "2026-06-06T11:25:24.000Z",
    "guest_info": {
      "name": "Jane Doe",
      "email": "jane.doe@example.com",
      "contact_number": "+15551234567"
    }
  }
  ```
* **Error Codes**:
  * `400 Bad Request`: `"guest info is required for guest checkout"`, `"guest name is required"`, `"guest email is required"`, `"guest contact number is required"`, `"quantity must be greater than zero"`, `"price cannot be negative"`.
  * `400 Bad Request / 409 Conflict`: `"insufficient stock"` (if products exceed stock thresholds).

### Create Authenticated Order (Protected)
* **Endpoint**: `POST /api/orders`
* **Security**: Required Bearer token.
* **Request Body**:
  ```json
  {
    "items": [
      {
        "variant_id": "var-501",
        "quantity": 2,
        "price": 24.99
      }
    ]
  }
  ```
* **Response `data` Shape**:
  ```json
  {
    "id": "order-778",
    "total": 49.98,
    "status": "pending",
    "created_at": "2026-06-06T11:25:24.000Z",
    "items": [
      {
        "variant_id": "var-501",
        "quantity": 2,
        "price": 24.99
      }
    ]
  }
  ```
* **Error Codes**:
  * `400 Bad Request`: `"order must contain at least one item"`, `"variant ID is required"`, `"quantity must be greater than zero"`, `"price cannot be negative"`.
  * `401 Unauthorized`.
  * `409 Conflict`: `"insufficient stock"`.

### List My Orders (Protected)
* **Endpoint**: `GET /api/orders`
* **Security**: Required Bearer token.
* **Response `data` Shape**:
  ```json
  {
    "orders": [
      {
        "id": "order-778",
        "total": 49.98,
        "status": "pending",
        "created_at": "2026-06-06T11:25:24.000Z",
        "items": [
          {
            "variant_id": "var-501",
            "quantity": 2,
            "price": 24.99
          }
        ]
      }
    ],
    "count": 1
  }
  ```
* **Error Codes**:
  * `401 Unauthorized`.

### Get My Order Details (Protected)
* **Endpoint**: `GET /api/orders/:id`
* **Security**: Required Bearer token.
* **Path Parameters**:
  * `id` (string): The Order ID.
* **Response `data` Shape**: Single Order object (identical structure to Create Order response).
* **Error Codes**:
  * `403 Forbidden`: `"you do not have access to this order"` (if attempting to fetch another user's order).
  * `404 Not Found`: `"order not found"`.

### Cancel Order (Protected)
* **Endpoint**: `POST /api/orders/:id/cancel`
* **Security**: Required Bearer token.
* **Response `data` Shape**: Updated single Order object (status: `canceled`).

---

## 7. Inventory Stock & Settings (Protected - Staff Only)

### Check Stock Status (Public)
* **Endpoint**: `GET /api/inventory/:variantID`
* **Path Parameters**:
  * `variantID` (string): Product variant ID.
* **Query Parameters**:
  * `location_id` (string, optional): Filter by location. Defaults to `"default"`.
* **Response `data` Shape**:
  ```json
  {
    "variant_id": "var-501",
    "location_id": "default",
    "quantity": 42
  }
  ```
* **Error Codes**:
  * `404 Not Found`: `"variant stock not found"`.

### List Inventory Levels (Protected)
* **Endpoint**: `GET /api/inventory`
* **Security**: Required Bearer token.
* **Query Parameters**:
  * `variant_id` (string, optional): Filter by variant.
  * `location_id` (string, optional): Filter by location.
  * `status` (string, optional): Filter by status (e.g. `low_stock`).
* **Response `data` Shape**:
  ```json
  {
    "items": [
      {
        "variant_id": "var-501",
        "location_id": "default",
        "quantity": 42,
        "low_stock_threshold": 10,
        "allow_backorders": false,
        "backorder_limit": 0,
        "is_low_stock": false
      }
    ]
  }
  ```
* **Error Codes**:
  * `401 Unauthorized`.

### List Active Stock Alerts (Protected)
* **Endpoint**: `GET /api/inventory/alerts`
* **Security**: Required Bearer token.
* **Response `data` Shape**:
  ```json
  {
    "alerts": [
      {
        "id": "alert-001",
        "type": "LOW_STOCK",
        "message": "Stock of variant TEE-BLK-L is low.",
        "variant_id": "var-501",
        "created_at": "2026-06-06T11:25:24.000Z",
        "is_read": false
      }
    ]
  }
  ```
* **Error Codes**:
  * `401 Unauthorized`.

### Update Stock Level (Protected)
* **Endpoint**: `POST /api/inventory/update`
* **Security**: Required Bearer token.
* **Request Body**:
  ```json
  {
    "variant_id": "var-501",
    "location_id": "default", // Optional. Defaults to "default"
    "quantity": 50 // New absolute quantity (minimum 0)
  }
  ```
* **Response `data` Shape**:
  ```json
  {
    "message": "stock updated"
  }
  ```
* **Error Codes**:
  * `400 Bad Request`: `"quantity cannot be negative"`.
  * `401 Unauthorized`.

### Configure Stock Thresholds & Settings (Protected)
* **Endpoint**: `POST /api/inventory/configure`
* **Security**: Required Bearer token.
* **Request Body**:
  ```json
  {
    "variant_id": "var-501",
    "location_id": "default", // Optional
    "quantity": 50, // Optional
    "low_stock_threshold": 10, // Optional
    "allow_backorders": true, // Optional
    "backorder_limit": 50 // Optional
  }
  ```
* **Response `data` Shape**:
  ```json
  {
    "message": "stock configured"
  }
  ```
* **Error Codes**:
  * `400 Bad Request`.
  * `401 Unauthorized`.

### Delete Stock Record (Protected)
* **Endpoint**: `DELETE /api/inventory/:variantID`
* **Security**: Required Bearer token.
* **Path Parameters**:
  * `variantID` (string): The product variant ID.
* **Query Parameters**:
  * `location_id` (string, optional): Location ID. Defaults to `"default"`.
* **Response `data` Shape**:
  ```json
  {
    "message": "stock deleted"
  }
  ```
* **Error Codes**:
  * `401 Unauthorized`.
  * `404 Not Found`: `"variant stock not found"`.

### Transfer Stock (Protected - Staff Only)
* **Endpoint**: `POST /api/inventory/transfer`
* **Security**: Required Bearer token & Staff role.
* **Request Body**:
  ```json
  {
    "variant_id": "var-501",
    "from_location_id": "default",
    "to_location_id": "secondary",
    "quantity": 5
  }
  ```
* **Response `data` Shape**: Message indicating success.

### Sync Stock Levels (Protected - Staff Only)
* **Endpoint**: `POST /api/inventory/sync`
* **Security**: Required Bearer token & Staff role.
* **Request Body**:
  ```json
  {
    "variant_id": "var-501",
    "location_id": "default",
    "quantity": 100
  }
  ```
* **Response `data` Shape**: Message indicating success.

### Reserve Variant Stock (Protected)
* **Endpoint**: `POST /api/inventory/:variantID/reserve`
* **Security**: Required Bearer token.
* **Request Body**:
  ```json
  {
    "location_id": "default",
    "quantity": 1
  }
  ```
* **Response `data` Shape**:
  ```json
  {
    "reservation_id": "res-999",
    "variant_id": "var-501",
    "location_id": "default",
    "quantity": 1,
    "expires_at": "2026-06-06T11:40:24.000Z"
  }
  ```

### Release Reservation (Protected)
* **Endpoint**: `DELETE /api/inventory/:variantID/reserve/:reservationID`
* **Security**: Required Bearer token.
* **Response `data` Shape**: Message indicating release success.

### Bulk Update Stock Levels (Protected - Staff Only)
* **Endpoint**: `POST /api/inventory/bulk-update`
* **Security**: Required Bearer token & Staff role.
* **Request Body**: Array of stock configurations.
* **Response `data` Shape**: Message indicating success.

### Adjust Stock Level (Protected - Staff Only)
* **Endpoint**: `POST /api/inventory/:variantID/adjust`
* **Security**: Required Bearer token & Staff role.
* **Request Body**:
  ```json
  {
    "location_id": "default",
    "adjustment": -2 // Can be positive or negative integer
  }
  ```
* **Response `data` Shape**: Message indicating adjustment success.

### Retrieve Low Stock Items (Protected - Staff Only)
* **Endpoint**: `GET /api/inventory/low-stock`
* **Security**: Required Bearer token & Staff role.
* **Response `data` Shape**: Array of low stock items.

### Export Stock Records (Protected - Staff Only)
* **Endpoint**: `GET /api/inventory/export`
* **Security**: Required Bearer token & Staff role.
* **Response `data` Shape**: CSV raw file or JSON export schema.

---

## 8. Admin Operations (Admin Role Required)

### Get Admin Dashboard Stats
* **Endpoint**: `GET /api/admin/dashboard`
* **Security**: Required Bearer token & Admin role.
* **Response `data` Shape**:
  ```json
  {
    "total_sales": 45320.50,
    "pending_orders": 12,
    "low_stock_skus": [
      "TEE-BLK-L",
      "SKU-4912"
    ],
    "active_users": 152
  }
  ```
* **Error Codes**:
  * `401 Unauthorized`.
  * `403 Forbidden`: `"forbidden: admin role required"`.

### List All System Orders
* **Endpoint**: `GET /api/admin/orders`
* **Security**: Required Bearer token & Admin role.
* **Response `data` Shape**:
  ```json
  {
    "orders": [
      {
        "id": "order-777",
        "customer_id": "", // Empty if guest order
        "guest_info": { // null/omitted if authenticated customer order
          "name": "Jane Doe",
          "email": "jane.doe@example.com",
          "contact_number": "+15551234567"
        },
        "items": [
          {
            "variant_id": "var-501",
            "quantity": 2,
            "price": 24.99
          }
        ],
        "total": 49.98,
        "status": "pending",
        "created_at": "2026-06-06T11:25:24.000Z"
      }
    ],
    "count": 1
  }
  ```
* **Error Codes**:
  * `401 Unauthorized`.
  * `403 Forbidden`: `"forbidden: admin role required"`.

### Get Any Order Details
* **Endpoint**: `GET /api/admin/orders/:id`
* **Security**: Required Bearer token & Admin role.
* **Path Parameters**:
  * `id` (string): The Order ID.
* **Response `data` Shape**: Single Order details object (same structure as above Admin order list entry).
* **Error Codes**:
  * `401 Unauthorized`.
  * `403 Forbidden`.
  * `404 Not Found`: `"order not found"`.

### Transition Order Status
* **Endpoint**: `POST /api/admin/orders/:id/transition`
* **Security**: Required Bearer token & Admin role.
* **Path Parameters**:
  * `id` (string): The Order ID.
* **Request Body**:
  ```json
  {
    "action": "pay" // Allowed action actions: "pay", "cancel", "ship", "complete"
  }
  ```
* **Allowed Transitions**:
  * `pending` + `"pay"` ➔ `paid`
  * `pending` + `"cancel"` ➔ `canceled`
  * `paid` + `"ship"` ➔ `shipped`
  * `shipped` + `"complete"` ➔ `done`
* **Response `data` Shape**: Updated single Order object.
* **Error Codes**:
  * `400 Bad Request`: `"invalid transition action"`.
  * `401 Unauthorized`.
  * `403 Forbidden`.
  * `404 Not Found`: `"order not found"`.

---

## 9. Notification Delivery

### Dispatch Custom User Notification
* **Endpoint**: `POST /api/notifications/send`
* **Security**: Required Bearer token.
* **Request Body**:
  ```json
  {
    "user_id": "60a89d70-c75c-4d33-911e-a4c3f58a36ef",
    "type": "email", // Options: "email", "sms", "webhook"
    "message": "Your order #order-778 has been shipped!"
  }
  ```
* **Response `data` Shape**:
  ```json
  {
    "id": "notif-091",
    "user_id": "60a89d70-c75c-4d33-911e-a4c3f58a36ef",
    "type": "email",
    "message": "Your order #order-778 has been shipped!",
    "sent_at": "2026-06-06T11:25:24.000Z",
    "status": "sent" // Options: "sent", "failed"
  }
  ```
  * `401 Unauthorized`.
  * `500 Internal Server Error`: Delivery engine/gateway failures.

---

## 10. Shipping Services

### Calculate Shipping Rates (Public)
* **Endpoint**: `POST /api/shipping/rates`
* **Request Body**:
  ```json
  {
    "weight_kg": 2.5,
    "destination": "123 Main St, Springfield"
  }
  ```
* **Response `data` Shape**: Array of shipping options and prices.

### Track Shipment (Public)
* **Endpoint**: `GET /api/shipping/track/:tracking_number`
* **Path Parameters**:
  * `tracking_number` (string): Shipping carrier tracking reference.
* **Response `data` Shape**:
  ```json
  {
    "tracking_number": "TRK123456",
    "status": "in_transit",
    "estimated_delivery": "2026-07-01T17:00:00Z"
  }
  ```

### Get Order Shipment details (Protected)
* **Endpoint**: `GET /api/shipping/order/:order_id`
* **Security**: Required Bearer token.
* **Path Parameters**:
  * `order_id` (string): Customer Order ID.
* **Response `data` Shape**: Shipment details matching the order.

### List All Shipments (Protected - Admin only)
* **Endpoint**: `GET /api/admin/shipping`
* **Security**: Required Bearer token & Admin role.
* **Response `data` Shape**: List of all system shipments.

### Create Shipment (Protected - Admin only)
* **Endpoint**: `POST /api/admin/shipping`
* **Security**: Required Bearer token & Admin role.
* **Request Body**:
  ```json
  {
    "order_id": "order-777",
    "carrier": "FedEx",
    "tracking_number": "TRK123456"
  }
  ```
* **Response `data` Shape**: Created shipment object.

### Update Shipment Status (Protected - Admin only)
* **Endpoint**: `PUT /api/admin/shipping/:id`
* **Security**: Required Bearer token & Admin role.
* **Path Parameters**:
  * `id` (string): Shipment ID.
* **Request Body**:
  ```json
  {
    "status": "delivered"
  }
  ```
* **Response `data` Shape**: Updated shipment object.

### Delete Shipment Record (Protected - Admin only)
* **Endpoint**: `DELETE /api/admin/shipping/:id`
* **Security**: Required Bearer token & Admin role.
* **Response `data` Shape**: Message confirming deletion.

---

## 11. Payment Processing

### Process Payment Webhook Callback (Public)
* **Endpoint**: `POST /api/payments/callback/:provider`
* **Path Parameters**:
  * `provider` (string): e.g., `stripe`, `paypal`
* **Request Body**: Raw provider payload.
* **Response `data` Shape**: Webhook process outcome.

### Create Payment checkout intent (Protected)
* **Endpoint**: `POST /api/payments/intent`
* **Security**: Required Bearer token.
* **Request Body**:
  ```json
  {
    "order_id": "order-778"
  }
  ```
* **Response `data` Shape**:
  ```json
  {
    "client_secret": "pi_123_secret_abc",
    "payment_intent_id": "pi_123"
  }
  ```

### List Payments (Protected)
* **Endpoint**: `GET /api/payments`
* **Security**: Required Bearer token.
* **Response `data` Shape**: Array of user payments.

### Get Payment by Order ID (Protected)
* **Endpoint**: `GET /api/payments/by-order/:orderID`
* **Security**: Required Bearer token.
* **Path Parameters**:
  * `orderID` (string): Order ID.
* **Response `data` Shape**: Payment details.

### List System Payments (Protected - Admin only)
* **Endpoint**: `GET /api/admin/payments`
* **Security**: Required Bearer token & Admin role.
* **Response `data` Shape**: Array of all system payments.

### Refund Payment (Protected - Admin only)
* **Endpoint**: `POST /api/admin/payments/refund`
* **Security**: Required Bearer token & Admin role.
* **Request Body**:
  ```json
  {
    "payment_id": "pay-101",
    "amount": 24.99
  }
  ```
* **Response `data` Shape**: Refund status.

### Get Payment Details (Protected - Admin only)
* **Endpoint**: `GET /api/admin/payments/:id`
* **Security**: Required Bearer token & Admin role.
* **Path Parameters**:
  * `id` (string): Payment ID.
* **Response `data` Shape**: Detailed payment record.

