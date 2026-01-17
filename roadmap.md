# MONTI – Development Roadmap

This document describes the rough development plan for **MONTI**, the live monitoring app for call centers.

---

## 1️⃣ Project Setup
- [x] Create GitHub repo (private)
- [x] Initialize Go backend project
- [x] Initialize frontend WebApp (Vite)
- [x] Create basic GitHub Actions workflow structure for CI/CD

---

## 2️⃣ Auth & Access (Local Development)
- [x] OIDC integration for WebApp + Backend (Keycloak local)
- [x] JWT token validation and claims parsing
- [x] Role extraction (Groups → Roles)
- [x] Backend middleware: Auth protection
- [x] Protected routes and login flow
- [x] WebSocket authentication with tokens

---

## 3️⃣ Data Model & Simulation
- [x] Database schema (Agents, Teams, Locations, Status)
- [x] Cache structure (Redis or in-memory)
- [x] Fake agent generator (2000 agents, random teams/locations)
- [x] API endpoints for streaming data

---

## 4️⃣ Backend Core
- [x] WebSocket service for live updates
- [x] Aggregation & grouping in cache
- [x] REST/GraphQL endpoints for initial data
- [x] Performance tests with 2000 agents

---

N## 5️⃣ Frontend
- [x] Dashboard basic layout
- [x] Implement WebSocket client
- [x] Grouped display of agents
- [x] Filter & sorting by teams / locations

---

## 6️⃣ Infrastructure & Deployment
- [ ] Configure AWS IAM Identity Center (production SSO)
- [ ] Update OIDC configuration for AWS (environment variables only)
- [ ] Create Terraform project structure (VPC, IAM, DB, ECS/Lambda)
- [ ] Remote state setup (S3 + DynamoDB)
- [ ] CI/CD pipeline: Build, Test, Docker Image, Deploy
- [ ] Test staging environment

---

## 7️⃣ Optimization & Monitoring
- [ ] Performance measurements (CPU, RAM, WebSocket traffic)
- [ ] Logging & metrics (CloudWatch / Prometheus)
- [ ] Optimize cache / grouping / WebSocket payload

---

## 8️⃣ Launch & Internal Rollout
- [ ] Define user accounts / roles
- [ ] Test access to dashboard for team
- [ ] Collect feedback & make minor adjustments
- [ ] Final deployment for all internal users

---

> 💡 **Tip:**
> - Start with **Backend + Fake Agents + Cache**, then WebSocket, then Frontend.
> - We use Keycloak for local auth development - switching to AWS IAM Identity Center for production requires only configuration changes (no code changes).
> - Set up Terraform & CI/CD in parallel, not at the end.
