# MONTI — Echtzeit-Monitoring für Contact Center

**Stack:** Go 1.23 / React 18 / WebSocket / Keycloak / AWS / Terraform
**Rolle:** Solo-Entwickler — Architektur, Backend, Frontend, Infrastruktur, Deployment

---

## Das Problem

Enterprise Contact Center auf Genesys Cloud nutzen native Monitoring-Tools, die APIs im 15–30-Sekunden-Takt pollen. Bei 2.000 Agents bedeutet das:

- **Veraltete Daten.** Supervisors sehen Agentenstatus, die eine halbe Minute alt sind. Ein „Ghost Agent" — jemand, dessen WebRTC-Verbindung abgebrochen ist, der aber weiterhin als verfügbar angezeigt wird — bleibt minutenlang unentdeckt und verschlechtert die Queue-Performance.
- **Blinde Flecken.** Native Dashboards zeigen Queue-Level-Aggregate, aber keine Anomalien auf Agent-Ebene: ungewöhnlich lange Nachbearbeitungszeiten, Auslastungsspitzen, Agents in Zwischenzuständen.
- **Kein Filtering.** Multi-Standort-Betriebe brauchen standortbezogene Views. Die eingebauten Tools unterstützen das nicht ohne manuelle Workarounds.

Supervisors flogen in Stoßzeiten blind. Calls stauten sich, weil das System dachte, Agents wären verfügbar — obwohl sie es nicht waren.

---

## Was ich gebaut habe

Eine Echtzeit-Monitoring-Plattform, die Polling durch eine persistente, event-getriebene Pipeline ersetzt. Jede Statusänderung und jeder Heartbeat fließt über WebSocket in ein Go-Backend, das aggregiert, anreichert und jede Sekunde einen vollständigen operativen Snapshot an alle verbundenen Supervisors sendet.

### Datenfluss

```
Agents (2.000 WebSocket-Verbindungen)
  → Heartbeats alle 2s + Statusänderungen bei Transition
    → Go Backend (AgentHub)
      → In-Memory State Tracker (RWMutex-geschützte Map)
      → 1-Sekunden-Aggregationsschleife
        → Snapshot: alle Agents + 16 Virtual Queues
          → RBAC-Filtering pro Client
            → Broadcast an Supervisor-Dashboards (WebSocket Hub)
```

### Was Supervisors sehen

- Live-Agent-Grid — farbcodiert nach Status, filterbar nach Abteilung/Standort
- Virtual-Queue-Panels — wartende Calls, durchschnittliche Wartezeit, Service Level pro Skill-Gruppe
- Alert-Indikatoren — Auslastung > 80%, Nachbearbeitung > 5 Min, Leerlauf > 16 Min
- Ghost-Agent-Erkennung — Agents werden nach 6 Sekunden als „stale" markiert (3 verpasste Heartbeats)

---

## Architekturentscheidungen und Tradeoffs

### 1. WebSocket überall statt SSE oder Polling

**Entscheidung:** Sowohl Agent-zu-Backend als auch Backend-zu-Frontend nutzen persistente WebSocket-Verbindungen.

**Warum:** Ich brauchte bidirektionale Kommunikation für die Agent-Verbindungen (Heartbeats hoch, Commands runter) und Sub-Sekunden-Latenz fürs Frontend. SSE würde für die Frontend-Hälfte funktionieren, hätte aber zwei Transport-Layer bedeutet. WebSocket überall hält das mentale Modell einfach.

**Tradeoff:** 2.000+ persistente Verbindungen auf einer einzelnen t3.small. Gelöst mit Goroutine-per-Connection in Go (günstig — ~50KB pro Goroutine) und gepufferten Channels, um Backpressure-Propagation zu verhindern.

### 2. Vollständiger Snapshot statt differenzieller Updates

**Entscheidung:** Jede Sekunde baut das Backend einen kompletten Snapshot (alle Agents + alle Queues) und broadcastet ihn an jeden Client.

**Warum:** Differenzielle Updates sind bandbreiteneffizienter, aber massiv komplexer — man braucht zuverlässiges Ordering, clientseitige Reconciliation und Recovery-Logik für verpasste Deltas. Ein vollständiger Snapshot mit ~200KB komprimiert liegt locker im WebSocket-Limit, und das Frontend wird trivial einfach: State ersetzen, neu rendern.

**Tradeoff:** Mehr Bandbreite pro Zyklus. Akzeptabel für <50 gleichzeitige Supervisors. Bei größerer Skalierung müsste man Delta-Compression einführen.

### 3. In-Memory State statt Datenbank im Hot Path

**Entscheidung:** Aller Agentenstatus lebt in einer einzelnen `map[string]*AgentInfo` hinter einem `RWMutex`. Kein Redis, keine Message Queue.

**Warum:** Der gesamte Datensatz passt in ~260MB. Ein Cache-Layer (Redis) oder ein Message Broker (NATS, Kafka) würde Latenz, operationale Komplexität und Failure Modes hinzufügen — für Daten, die von Natur aus ephemer sind. Bei einem Prozess-Neustart registrieren sich Agents innerhalb von Sekunden neu.

**Tradeoff:** Single-Process, Single-Node. Kein horizontales Scaling. Das ist okay für das Ziel-Deployment (ein Contact Center, eine Backend-Instanz). Für Multi-Region oder >10k Agents würde ich nach Abteilung sharden und NATS für Cross-Shard-Aggregation einsetzen.

### 4. Agent-Simulation als First-Class-Komponente

**Entscheidung:** Ein vollständiger Agent-Simulator (AgentSim) als separater Go-Service, der realistischen Contact-Center-Traffic erzeugt — State Machines mit konfigurierbaren Übergangszeiten, Call-Generierung mit Peak-Hour-Modellierung und eine Control-API zum Skalieren.

**Warum:** Man kann kein Echtzeit-Monitoring-System entwickeln oder demonstrieren, ohne Echtzeitdaten. AgentSim lässt mich 1.000 Agents mit realistischen Verhaltensmustern in Sekunden hochfahren. Es dient gleichzeitig als Lasttesttool — ich kann verifizieren, dass das Backend 1.000 Heartbeats/Sekunde verarbeitet, ohne dass die Aggregationsschleife 100ms überschreitet.

**Tradeoff:** Ein weiterer Service zum Warten. Lohnt sich — die Entwicklung wurde 10x schneller, und die Demo läuft immer live.

### 5. Keycloak für Auth statt eigener JWT-Lösung

**Entscheidung:** Vollständiger OIDC-Flow mit Keycloak — Authorization Code + PKCE, JWT-Validierung via JWKS, rollenbasierte Zugriffskontrolle, Business-Unit-Filterung über Group Claims.

**Warum:** Enterprise Auth ist nichts, was man selbst bauen will. Keycloak liefert User-Management, gruppenbasierte Zugriffskontrolle und Token-Lifecycle out of the box. Business-Unit-Filterung (SGB/NGB/RGB) mapped direkt auf Keycloak Groups, sodass Supervisors nur Agents in ihrer Region sehen.

**Tradeoff:** Keycloak ist schwergewichtig (~500MB RAM). Für ein Portfolio-Projekt würde ein einfacheres Auth-Setup reichen. Aber es spiegelt reale Enterprise-Deployments wider — und genau das war der Punkt.

### 6. Terraform + geplante EC2 statt Always-On

**Entscheidung:** Die Produktions-EC2-Instanz läuft nach Zeitplan — EventBridge startet sie um 13:45 CET und stoppt sie um 16:05 CET an Werktagen. Gesamtkosten: ~20€/Monat.

**Warum:** Das ist ein Portfolio-Projekt, kein umsatzgenerierendes SaaS. Ich wollte produktionsreife Infrastruktur (IaC, TLS, CDN, Monitoring) ohne produktionsreife Rechnungen. Das Zeitfenster gibt jeden Werktagnachmittag eine zuverlässige Demo-Möglichkeit.

**Tradeoff:** Nicht 24/7 erreichbar. Das Frontend fängt das elegant ab — eine `ScheduleGate`-Komponente prüft die Backend-Verfügbarkeit und zeigt außerhalb des Fensters eine Offline-Seite.

---

## Ergebnisse

| Metrik | Native Tools | MONTI |
|--------|-------------|-------|
| Datenaktualität | 15–30s Polling | **1s Broadcast-Zyklus** |
| Ghost-Agent-Erkennung | Minuten (manuell) | **6 Sekunden (automatisch)** |
| Aggregationslatenz | N/A | **<50ms für 2.000 Agents** |
| Anomalie-Alerts pro Agent | Nicht verfügbar | **Echtzeit (Auslastung, NBZ, Leerlauf)** |
| Business-Unit-Filterung | Manuell | **Automatisch via OIDC Groups** |
| WebSocket-Durchsatz | N/A | **1.000 Heartbeats/Sek. sustained** |

---

## Was ich gelernt habe

**Gos Concurrency-Modell ist das richtige Werkzeug dafür.** Goroutine-per-Connection mit Channel-basierter Koordination macht die nebenläufige Architektur geradlinig. Das gleiche System in Node.js wäre bei dieser Verbindungsanzahl ein Event-Loop-Scheduling-Albtraum geworden.

**Vollständige Snapshots schlagen clevere Diffs — bis sie es nicht mehr tun.** Für diese Größenordnung ist es einfacher und zuverlässiger, jede Sekunde die komplette Welt zu senden, als differenziellen State zu managen. Aber ich sehe genau, wo das kippt (~100 Clients oder ~10k Agents), und ich kenne den Migrationspfad.

**Simuliere deine Datenschicht früh.** AgentSim vor dem Frontend zu bauen hat sich enorm ausgezahlt. Jede Dashboard-Komponente wurde von Tag eins an gegen realistische Daten entwickelt, was Edge Cases aufgedeckt hat (Agents in Übergangszuständen, leere Queues, gleichzeitige Statuswechsel), die ich mit Mock-Daten nie gefunden hätte.

**Infrastructure as Code ist nicht optional, auch nicht bei Nebenprojekten.** Das Terraform-Setup hat einen Tag gekostet. Es hat mir Dutzende Stunden in reproduzierbaren Deployments gespart — und demonstriert Infrastrukturkompetenz, die ein README allein nicht vermitteln kann.

---

## Links

- **Live-Demo:** [monti.dennisdiepolder.com](https://monti.dennisdiepolder.com) *(Werktags 14:00–16:00 CET)*
- **Quellcode:** Auf Anfrage verfügbar

---

*Gebaut von Dennis Diepolder. Fragen willkommen.*
