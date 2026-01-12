# MONTI – Development Roadmap

This document describes the rough development plan for **MONTI**, the live monitoring app for call centers.

---

## 1️⃣ Project Setup
- [x] Create GitHub repo (private)
- [x] Initialize Go backend project
- [x] Initialize frontend WebApp (Vite)
- [x] Create basic GitHub Actions workflow structure for CI/CD

---

## 2️⃣ Auth & Access
- [ ] Configure AWS IAM Identity Center (internal)
- [ ] OIDC integration for WebApp + Backend
- [ ] JWT claims mapping (Groups → Roles)
- [ ] Backend middleware: Check access (`can(user, action)`)

---

## 3️⃣ Data Model & Simulation
- [ ] Database schema (Agents, Teams, Locations, Status)
- [ ] Cache structure (Redis or in-memory)
- [ ] Fake agent generator (2000 agents, random teams/locations)
- [ ] API endpoints for streaming data

---

## 4️⃣ Backend Core
- [ ] WebSocket service for live updates
- [ ] Aggregation & grouping in cache
- [ ] REST/GraphQL endpoints for initial data
- [ ] Performance tests with 2000 agents

---

## 5️⃣ Frontend
- [ ] Dashboard basic layout (Teams, Locations, Status)
- [ ] Implement WebSocket client
- [ ] Grouped display of agents
- [ ] Filter & sorting by teams / locations

---

## 6️⃣ Infrastructure & Deployment
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
> - IAM Identity Center integration early on, otherwise you'll have to change everything later.
> - Set up Terraform & CI/CD in parallel, not at the end.
