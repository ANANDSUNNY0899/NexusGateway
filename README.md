# NexusGateway

**Nexus Gateway** is an intelligent AI Middleware designed to slash LLM costs by up to 90% and reduce latency by 100x. It uses **Semantic Caching** (Vector Database) to understand the *meaning* of user queries, not just the keywords.

> **Status:** Production Ready (Redis + Pinecone + Rate Limiting)

##  Key Features

*   **Semantic Caching:** Uses **OpenAI Embeddings** & **Pinecone** to detect similar questions (e.g., "Tea recipe" = "How to make tea") and serves cached answers.
*   **Zero-Latency Mode:** Serves cached responses in **<50ms**.
*   **Bankruptcy Protection:** Built-in **Rate Limiting** (Token Bucket) to prevent API abuse and cost spikes.
*   **Multi-Layer Storage:**
    *   **L1:** Redis (Hot Cache for exact matches & stats).
    *   **L2:** Pinecone (Vector Store for semantic matches).
*   **Analytics Engine:** Tracks cache hits, misses, and estimated cost savings.

## Tech Stack

*   **Core:** Go (Golang)
*   **Vector DB:** Pinecone (Serverless)
*   **Cache:** Redis (Upstash)
*   **AI:** OpenAI (GPT-3.5 + text-embedding-3-small)

## Project Structure

```bash
NexusGateway/
├── config/         # Environment & Key Management
├── handler/
│   ├── chat.go       # Main Logic (Orchestrator)
│   ├── embedding.go  # OpenAI Vector Generation
│   ├── pinecone.go   # Vector Database Operations
│   ├── redis.go      # Caching & Stats
│   ├── middleware.go # Rate Limiting Security
│   └── stats.go      # Analytics Endpoint
├── main.go         # Server Entry Point
└── go.mod          # Dependencies
Getting Started
Prerequisites
Go 1.21+
OpenAI API Key
Pinecone API Key & Host URL
Redis Connection String
Installation
Clone the repository:
code
Bash
git clone https://github.com/YOUR_USERNAME/NexusGateway.git
cd NexusGateway
Set Environment Variables (Windows CMD):
code
Cmd
set OPENAI_API_KEY=sk-...
set REDIS_URL=rediss://default:pass@url:6379
set PINECONE_API_KEY=pcsk_...
set PINECONE_HOST=index-name.svc.pinecone.io
set PORT=8080
Run the Server:
code
Bash
go run main.go
 API Usage
1. Chat Completion (Smart Cache)
POST /api/chat
code
JSON
{
  "message": "How do I optimize a SQL query?"
}
Response Headers:
X-Cache: HIT-SEMANTIC (Served from Pinecone)
X-Cache: MISS (Served from OpenAI)
2. Analytics
GET /api/stats
code
JSON
{
    "total_requests": "150",
    "cache_hits": "45",
    "cache_misses": "105",
    "money_saved_est": "$0.20"
}
How Semantic Search Works
User sends text: "How to fix a flat tire?"
Gateway converts text to a 1536-dimensional vector.
Queries Pinecone for similar vectors (Threshold > 0.70).
If Found: Returns stored answer immediately.
If New: Calls OpenAI, returns answer, and saves vector for future users.

Built by SUNNY ANAND