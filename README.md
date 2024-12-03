# Comment Toxicity Filter

A simple HTTP service I built for a friend of mine that uses Google's Gemini AI to classify comments as harmful or non-harmful. It returns either `true` for harmful comments or `false` for safe ones.
It uses Gemini-1.5-Pro-Flash with a carefully crafted system prompt to classify comments. 

Gemini-1.5-Pro-Flash is currently (December 2024) free with the following limits:
- 15 requests per minute (RPM)
- 1 million tokens per minute (TPM)
- 1,500 requests per day (RPD)

While both methods work, please use the JSON method for your production system. The URL method is mainly there for testing and compatibility with my friends current setup.

## Setup

1. Copy `docker-compose.yml.example` to `docker-compose.yml`
2. Get your Gemini API key from Google [here](https://aistudio.google.com/app/apikey)
3. Add your API key and optional environment variables to the `docker-compose.yml`:
   ```yaml
   #required
   GEMINI_API_KEY=your-api-key-here

   # optional
   CUSTOM_GEMINI_MODEL=gemini-1.5-flash  # Optional: Use gemini-1.5-flash-8b for faster but less accurate results
   CUSTOM_REQUEST_TIMEOUT_SECONDS=30     # Optional: Change request timeout (must be positive integer)
   HTTP_LISTEN_ADDR=0.0.0.0:3343         # Optional: Change listen address and port.
   ```
4. Optionally change the port by modifying `HTTP_LISTEN_ADDR` and the `ports` mapping in `docker-compose.yml`
5. Run the service:
   ```bash
   docker compose up -d
   ```

Note: Both gemini-1.5-flash and gemini-1.5-flash-8b currently have the same free tier limits. The flash model is used by default as it provides better results, but if speed is a priority, the 8b variant is faster.

The service will be available at port 3343 by default.

## Usage

There are two ways to use the service:

### 1. Simple URL Method
Good for quick testing and compatibility with the existing system, but limited to shorter comments due to URL length restrictions (max. 2000 characters per URL):
```bash
curl "http://localhost:3343/block/this%20is%20a%20test%20comment"
```

### 2. JSON Method (Recommended)
Handles longer comments and special characters properly:
```bash
curl -X GET -H "Content-Type: application/json" \
     -d '{"comment": "this is a test comment"}' \
     http://localhost:3343/block
```

## Response Format

- URL method returns: `"True"` or `"False"`
- JSON method returns: `{"block": true}` or `{"block": false}`

The URL method exists for backward compatibility with the existing system and for quick testing. The JSON method is recommended for all new implementations and should be used for the production system.
