# Project MONTI

**MONTI** is a live monitoring app, specifically developed for call centers.

## Goal
The app should deliver **high performance** and be able to display **over 2000 agents live**.

## Architecture
- **Frontend:** WebApp for optimal usability
- **Backend:** Go
- **Data:** Simulation of 2000 fake agents from different teams and locations

## Data Processing
- All agent data is stored in a **database**
- Simultaneously, data is collected and grouped in a **cache**
- The WebApp receives only **filtered groupings** instead of individual agents to ensure performance
- WebSocket requests are thereby **reduced**, keeping the app performant

## Authentication & Security
- **AWS Identity Center** centrally for authentication
- Data access is controlled **tenant- and location-based**
- No external access without valid identity

## CI/CD
- Repository on **GitHub** → Using **GitHub Actions** for Continuous Integration and Deployment
- Workflow includes:
  - Build & Lint of Go backend
  - Testing all components
  - Creating Docker images (optional)
  - Deployment to AWS (Lambda / ECS / EC2)
- Terraform infrastructure can also be managed via GitHub Actions

## Infrastructure
- Provisioning of cloud resources via **Terraform**
- Advantages:
  - Modular & reusable (modules for VPC, DB, IAM, ECS/Lambda)
  - Multi-cloud capable, if needed in the future
  - Remote State Management (S3 + DynamoDB)
  - Easy CI/CD integration via GitHub Actions
- Resources managed via Terraform:
  - **IAM Identity Center** for auth & access rights
  - **Backend Infrastructure** (Lambda / ECS / EC2)
  - **Database & Cache** (RDS / DynamoDB / Redis)
  - **Network & Security** (VPC, Subnets, Security Groups)
