# 006: Lightweight hexagonal architecture

The project is organized around a small, technology-neutral core and adapters on the outside. This keeps the journaling logic independent from Google APIs, cookie handling, and the web layer while still allowing deployment to local and Google-backed storage transparently.
