# 002: Use encrypted signed cookie sessions

The app keeps session state in an encrypted, signed HttpOnly cookie instead of a database-backed server-side session. This matches the requirement to avoid storing journal content server-side while still carrying the provider connection and refresh token through the app lifecycle.
