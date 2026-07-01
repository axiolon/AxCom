# Ecom Engine - Postman API Documentation

This directory contains consolidated Postman collections and environment setups to test all modules of the Ecom Engine.

## Files included:
- **Ecom_Engine.postman_collection.json**: The complete unified collection containing all Auth, Cart, Catalog, Inventory, Orders, and Payments endpoints grouped neatly in directories.
- **Ecom_Engine.postman_environment.json**: Pre-configured environment file pointing to local server settings.
- **modules/**: Individual collection files for each subsystem, if you prefer importing only specific modules.

## How to Import & Setup:
1. Open Postman.
2. Click **Import** in the top-left corner.
3. Select and import both **Ecom_Engine.postman_collection.json** and **Ecom_Engine.postman_environment.json**.
4. In the top-right environment selector dropdown, select **Ecom Engine - Local**.
5. Run the **Login User** or **Register User** endpoint under the **Auth** module. Upon successful login, the environment variables `access_token` and `refresh_token` will automatically be updated by the post-response script, authenticating all subsequent requests.
