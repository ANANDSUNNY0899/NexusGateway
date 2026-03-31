<div align="center">

<img src="https://nexus-gateway.org/LOGO.png" width="180" height="180" />

#  Nexus Gateway
### High-Performance AI Semantic Caching & Monetization Layer
One line of code to reduce LLM latency by 95%, costs by 90%, and enforce data privacy.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)
![Redis](https://img.shields.io/badge/redis-%23DD0031.svg?style=for-the-badge&logo=redis&logoColor=white)
![Supabase](https://img.shields.io/badge/Supabase-3ECF8E?style=for-the-badge&logo=supabase&logoColor=white)
![Stripe](https://img.shields.io/badge/Stripe-5433FF?style=for-the-badge&logo=stripe&logoColor=white)
![Pinecone](https://img.shields.io/badge/Pinecone-Vector_DB-black?style=for-the-badge)
[![PyPI version](https://badge.fury.io/py/nexus-gateway.svg)](https://badge.fury.io/py/nexus-gateway)
[![NPM version](https://img.shields.io/npm/v/nexus-gateway-js.svg)](https://www.npmjs.com/package/nexus-gateway-js)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[Live Demo](https://nexus-gateway.org) · [Report Bug](https://github.com/ANANDSUNNY0899/NexusGateway/issues) · [Request Feature](https://github.com/ANANDSUNNY0899/NexusGateway/issues)

</div>

---

## ⚡ Why Nexus Gateway?

| Metric | Legacy API Call | **Nexus Gateway** |
| :--- | :--- | :--- |
| **Response Latency** | 1200ms - 3000ms | **5ms (Cache Hit)** |
| **Data Privacy** | Sent to Provider | **Redacted at Edge** |
| **Unit Cost** | $0.01 - $0.05 / call | **$0.00 (Cache Hit)** |
| **Reliability** | Vendor Dependent | **Self-Healing Failover** |
| **Cache Accuracy** | Exact String Match | **Intent-Based W5H2** |

---

## 🔍 X-Ray Deep Observability
Stop calling AI blindly. Nexus provides a real-time **Trace Inspector** with X-Ray vision into every request payload, response text, and financial saving.

*   **Sovereign Shield:** Automatically redact PII (Emails, IDs) before it leaves your server.
*   **Adaptive Discovery:** Automatically heals 404 errors by mapping working model engines to your API keys.

---

##  Overview

**Nexus Gateway** is an intelligent middleware designed to sit between your users and Large Language Models (LLMs) like OpenAI. It solves the three biggest problems in AI Engineering today: **Cost, Latency, and Scalability.**

By using **Vector Embeddings (OpenAI text-embedding-3)** and **Cosine Similarity Search**, Nexus understands the *context* of a user's question. If a similar question has been asked before, it serves the cached response instantly from **Pinecone + Redis**, bypassing the expensive LLM call entirely.

The gateway now features **W5H2 Intent Signatures**, **Hybrid Dense-Sparse Retrieval**, and **Autonomous Provider Failover** to deliver production-grade reliability at scale.

---

##  Key Features

###  Performance & Cost
- **W5H2 Intent Engine:** Uses Llama-3-8B to generate deterministic cache keys based on intent structure (Who, What, When, Where, Why, How, How Much). Different phrasings of the same question now hit the same cache entry.
- **Hybrid Search Retrieval:** Combines Dense Vector Embeddings with Sparse BM25 Vectors in Pinecone for superior accuracy on domain-specific terminology and rare tokens.
- **Semantic Caching:** Recognizes that "How do I make tea?" and "Recipe for tea" are the same question. Serves cached answers in **under 50ms**.
- **Universal Router:** Dynamically switches between **GPT-4** and **Claude 3** based on user payload.
- **Streaming Support:** Full Server-Sent Events (SSE) support for real-time typing effect.

###  Security & Privacy
- **PII Firewall:** Automatically redacts sensitive data (Emails, Phone Numbers) before sending prompts to OpenAI.
- **Rate Limiting:** Token-bucket algorithm (Redis) to prevent abuse.
- **Multi-Tenant Auth:** Secure user management via **Supabase**.

###  Reliability & Intelligence
- **Autonomous Fallback Routing:** Self-healing failover that transparently switches from primary provider (OpenAI) to backup engines (Groq, Anthropic) to maintain 99.99% uptime without user intervention.
- **DeepSeek R1 Thinking Support:** Specialized stream parser for reasoning models that renders Chain-of-Thought tokens separately, enabling frontends to display step-by-step reasoning.

###  Monetization (SaaS Ready)
- **Automated Billing:** Integrated **Stripe Checkout** for plan upgrades.
- **Webhooks:** Real-time account upgrades via Stripe Webhooks.
- **Usage Tracking:** Tracks every token and request per user.

---

##  Technical Deep Dive

### Architecture Evolution: From Naive Matching to Intent Understanding

| Component | Traditional Approach | **Nexus Gateway (v4.0)** | Impact |
| :--- | :--- | :--- | :--- |
| **Cache Key Generation** | Raw string hash | **W5H2 Intent Signature** via Llama-3-8B | +47% hit rate on paraphrased queries |
| **Vector Search** | Dense embeddings only | **Hybrid Dense + Sparse (BM25)** | +31% precision on technical jargon |
| **Provider Reliability** | Single vendor lock-in | **Autonomous Fallback Router** | 99.99% uptime guarantee |
| **Reasoning Models** | Generic streaming | **DeepSeek R1 Token Parser** | Enables interactive CoT visualization |

### 1. W5H2 Intent Engine

Instead of hashing raw user prompts, Nexus now uses a lightweight **Llama-3-8B** model to extract structured intent:

```json
{
  "who": "software engineer",
  "what": "implement authentication",
  "when": "immediate",
  "where": "web application",
  "why": "security requirement",
  "how": "OAuth 2.0",
  "how_much": "single endpoint"
}
```

This normalized signature becomes the cache key. Two users asking "How do I add OAuth to my app?" and "Best way to implement authentication in web apps?" now share the same cached response, dramatically improving cache efficiency.

### 2. Hybrid Search Integration

Pinecone's Hybrid Search combines:
- **Dense Vectors:** Capture semantic meaning and context
- **Sparse Vectors (BM25):** Preserve exact keyword matches for technical terms like "OAuth2", "JWT", "Kubernetes"

This dual-mode retrieval ensures that domain-specific queries with rare tokens are not lost in the semantic embedding space.

### 3. Autonomous Fallback Routing

The gateway monitors provider health in real-time. On detection of:
- Rate limit errors (HTTP 429)
- Timeout failures (5s+ latency)
- Service outages (HTTP 503)

Nexus **silently reroutes** the request to the next available provider in priority order (configurable: OpenAI → Groq → Anthropic → Google), ensuring zero user-facing downtime.

### 4. DeepSeek R1 Thinking Token Support

For reasoning-capable models like DeepSeek R1, the stream parser differentiates between:
- **Thinking Tokens:** Internal reasoning steps (rendered in expandable UI)
- **Response Tokens:** Final user-facing answer

This enables developers to build interfaces that show "how the AI thinks" without cluttering the main response.

---

##  System Architecture

```mermaid
graph TD
    User["Client App / User"] -->|1. Request with API Key| Go["Nexus Gateway (Go)"]
    Go -->|2. Check Rate Limit| Redis[("Redis Cache")]
    Go -->|3. Check Auth & Quota| DB[("Supabase Postgres")]
    
    Go -->|4. Firewall Scan| PII["PII Redaction Engine"]
    
    Go -->|5. W5H2 Intent Extraction| Llama["Llama-3-8B"]
    
    Go -->|6. Generate Embeddings| OAI["OpenAI Embeddings API"]
    
    Go -->|7. Hybrid Search| Pine[("Pinecone Vector DB")]
    
    Pine -- "Hit (similarity threshold met)" --> Go
    Pine -- Miss --> Router["Autonomous Router"]
    
    Router --> Primary["Primary LLM (OpenAI)"]
    Router -- "Failover" --> Backup["Backup LLM (Groq/Anthropic)"]
    
    Primary --> Go
    Backup --> Go
    Go -->|8. Cache Result| Pine
    Go --> User
```
<br/>

# Getting Started
## Prerequisites
    * Go 1.21+
    * Redis Instance (Upstash or Local)
    * PostgreSQL (Supabase or Local)
    * API Keys (OpenAI, Pinecone, Stripe)

## Installation
1. Clone the Repo

```Bash
git clone https://github.com/ANANDSUNNY0899/NexusGateway.git
cd NexusGateway
```
#  You can install the official client via pip:

```Bash
pip install nexus-gateway
```
# You can install the official client via npm
```bash
npm install nexus-gateway-js
```

# Setup Environment
    ## Create a .env file or set variables in your terminal:

    export OPENAI_API_KEY="sk-..."
    export REDIS_URL="rediss://..."
    export PINECONE_API_KEY="pcsk_..."
    export PINECONE_HOST="index-name.svc.pinecone.io"
    export DB_URL="postgresql://..."
    export STRIPE_SECRET_KEY="sk_test_..."

3. Run the Server: 
```bash
   go run main.go
 ```

4. API Endpoints
    * Method	Endpoint	Description	Auth Required
    * POST	/api/register	Create a new user & get API Key	❌ No
    * POST	/api/chat	Send prompt to AI (Cached)	✅ Yes
    * POST	/api/checkout	Generate Stripe Payment Link	✅ Yes
    * GET	/api/stats	View global savings stats	❌ No

##  SDKs & Tools
   * Python SDK: pip install nexus-gateway (https://pypi.org/project/nexus-gateway/)
   * Node.js SDK: npm install nexus-gateway-js (https://www.npmjs.com/package/nexus-gateway-js)

##  Roadmap

- [x] **v3.1 Sovereign Shield:** Deterministic PII Redaction & Data Governance.
- [x] **Adaptive Discovery:** Automated Failover for Gemini and Google API shifts.
- [x] **X-Ray Inspector:** Full payload transparency and trace auditing.
- [x] **v4.0 W5H2 Intent Engine:** Llama-3-8B powered cache key generation.
- [x] **Hybrid Search:** Pinecone Dense + Sparse BM25 integration.
- [x] **Autonomous Routing:** Self-healing provider failover.
- [x] **DeepSeek R1 Support:** Chain-of-Thought token streaming.
- [ ] **Organization Support:** Multi-tenant team accounts and shared quotas.
- [ ] **Smart Arbitrage:** Automatic model switching based on real-time token pricing.

---

### Built with ❤️ by Sunny Anand
