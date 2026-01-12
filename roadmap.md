# MONTI – Development Roadmap ✅

Dieses Dokument beschreibt den groben Entwicklungsplan für **MONTI**, die Live-Monitoring-App für Callcenter.

---

## 1️⃣ Projekt Setup
- [ ] GitHub Repo anlegen (privat)  
- [ ] Go Backend Projekt initialisieren  
- [ ] Frontend WebApp (React / Astro / Next.js) initialisieren  
- [ ] GitHub Actions Workflow Grundgerüst für CI/CD anlegen  

---

## 2️⃣ Auth & Zugriff
- [ ] AWS IAM Identity Center konfigurieren (intern)  
- [ ] OIDC Integration für WebApp + Backend  
- [ ] JWT-Claims Mapping (Groups → Roles)  
- [ ] Backend Middleware: Zugriff prüfen (`can(user, action)`)  

---

## 3️⃣ Datenmodell & Simulation
- [ ] Datenbank Schema (Agents, Teams, Standorte, Status)  
- [ ] Cache-Struktur (Redis oder In-Memory)  
- [ ] Fake Agent Generator (2000 Agents, random Teams/Standorte)  
- [ ] API Endpoints zum Streamen der Daten  

---

## 4️⃣ Backend Core
- [ ] WebSocket Service für Live Updates  
- [ ] Aggregation & Gruppierung im Cache  
- [ ] REST/GraphQL Endpoints für initiale Daten  
- [ ] Performance Tests mit 2000 Agents  

---

## 5️⃣ Frontend
- [ ] Dashboard Grundlayout (Teams, Standorte, Status)  
- [ ] WebSocket Client implementieren  
- [ ] Gruppierte Darstellung der Agents  
- [ ] Filter & Sortierung nach Teams / Standorten  

---

## 6️⃣ Infrastruktur & Deployment
- [ ] Terraform Projektstruktur anlegen (VPC, IAM, DB, ECS/Lambda)  
- [ ] Remote State Setup (S3 + DynamoDB)  
- [ ] CI/CD Pipeline: Build, Test, Docker Image, Deploy  
- [ ] Staging Environment testen  

---

## 7️⃣ Optimierung & Monitoring
- [ ] Performance-Messungen (CPU, RAM, WebSocket Traffic)  
- [ ] Logging & Metrics (CloudWatch / Prometheus)  
- [ ] Optimierung Cache / Gruppierung / WebSocket Payload  

---

## 8️⃣ Launch & Internal Rollout
- [ ] User Accounts / Roles definieren  
- [ ] Testzugriff auf Dashboard für Team  
- [ ] Feedback sammeln & kleine Anpassungen  
- [ ] Final Deployment für alle internen Nutzer  

---

> 💡 **Tipp:**  
> - Beginne mit **Backend + Fake Agents + Cache**, dann WebSocket, danach Frontend.  
> - IAM Identity Center Integration frühzeitig, sonst musst du später alles ändern.  
> - Terraform & CI/CD parallel aufsetzen, nicht erst am Ende.
