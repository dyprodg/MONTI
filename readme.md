# Projekt MONTI

**MONTI** ist eine Live-Monitoring-App, speziell entwickelt für Callcenter.

## Ziel
Die App soll eine **hohe Performance** liefern und **über 2000 Agents live** darstellen können.

## Architektur
- **Frontend:** WebApp für optimale Usability  
- **Backend:** Go  
- **Daten:** Simulation von 2000 Fake-Agents aus unterschiedlichen Teams und Standorten  

## Datenverarbeitung
- Alle Agent-Daten werden in einer **Datenbank** gespeichert  
- Gleichzeitig werden sie in einem **Cache** gesammelt und gruppiert  
- Die WebApp erhält nur **gefilterte Gruppierungen**, statt einzelner Agents, um die Performance zu sichern  
- WebSocket-Anfragen werden dadurch **reduziert**, die App bleibt performant

## Authentifizierung & Sicherheit
- **AWS Identity Center** zentral für Authentifizierung  
- Zugriff auf Daten wird **mandanten- und standortbasiert** gesteuert  
- Kein externer Zugriff ohne gültige Identity

## CI/CD
- Repository auf **GitHub** → Verwendung von **GitHub Actions** für Continuous Integration und Deployment  
- Workflow umfasst:
  - Build & Lint des Go-Backends  
  - Testen aller Komponenten  
  - Erstellung von Docker-Images (optional)  
  - Deployment zu AWS (Lambda / ECS / EC2)  
- Terraform-Infrastruktur kann ebenfalls über GitHub Actions verwaltet werden

## Infrastruktur
- Bereitstellung der Cloud-Ressourcen über **Terraform**  
- Vorteile:
  - Modular & wiederverwendbar (Module für VPC, DB, IAM, ECS/Lambda)  
  - Multi-Cloud-fähig, falls zukünftig nötig  
  - Remote State Management (S3 + DynamoDB)  
  - Einfache CI/CD Integration via GitHub Actions  
- Ressourcen, die über Terraform verwaltet werden:
  - **IAM Identity Center** für Auth & Zugriffsrechte  
  - **Backend Infrastruktur** (Lambda / ECS / EC2)  
  - **Datenbank & Cache** (RDS / DynamoDB / Redis)  
  - **Netzwerk & Security** (VPC, Subnets, Security Groups)
