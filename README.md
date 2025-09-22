# Cotizador Online

A web-based quote calculator application built with Go.

## Access

The application can be accessed from the web at: https://cotizador-demo.erp.homme.ar/

## Local Development

To run the application locally:

1. Ensure you have Go installed.
2. Clone the repository.
3. Run `go run main.go`.

## Docker

You can also run the application using Docker:

```bash
docker build -t cotizador-online .
docker run -p 8080:8080 cotizador-online
```

## Project Structure

- `main.go`: Main application file
- `templates/`: HTML templates
- `static/`: Static assets (CSS, JS, etc.)
- `Dockerfile`: Docker configuration