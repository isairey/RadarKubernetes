<div align="center">

<img width="220" src="https://cdn-icons-png.flaticon.com/512/5968/5968242.png" />

# ☸️ Radar

### Plataforma moderna de visualización y gestión para Kubernetes 🚀

<p align="center">
  <b>Radar</b> es una interfaz moderna y open source para Kubernetes que permite visualizar topologías, monitorear tráfico, administrar recursos, Helm y GitOps desde una sola herramienta local-first, rápida y sin dependencias cloud.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go&logoColor=white">
  <img src="https://img.shields.io/badge/Kubernetes-UI-326CE5?style=for-the-badge&logo=kubernetes&logoColor=white">
  <img src="https://img.shields.io/badge/License-Apache_2.0-blue?style=for-the-badge">
  <img src="https://img.shields.io/github/stars/skyhook-io/radar?style=for-the-badge&logo=github&color=yellow">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Topology-Visualization-7B61FF?style=flat-square">
  <img src="https://img.shields.io/badge/Helm-Management-0F1689?style=flat-square&logo=helm">
  <img src="https://img.shields.io/badge/GitOps-FluxCD%20%26%20ArgoCD-orange?style=flat-square">
  <img src="https://img.shields.io/badge/MCP-AI%20Integration-FF6B6B?style=flat-square">
  <img src="https://img.shields.io/badge/Real--Time-Monitoring-00C853?style=flat-square">
</p>

</div>

---

# 📖 Descripción

Radar es una poderosa herramienta de observabilidad y administración para clusters Kubernetes.  
Está diseñada bajo el enfoque **local-first**, funcionando directamente desde tu computadora sin depender de servicios cloud ni instalar componentes adicionales dentro del cluster.

La plataforma ofrece:

- Visualización interactiva de topologías
- Monitoreo de tráfico en tiempo real
- Gestión de Helm Releases
- Integración con FluxCD y ArgoCD
- Timeline de eventos
- Navegador de recursos Kubernetes
- Soporte MCP para IA
- Gestión TLS
- Cost Insights
- Cluster Audit

Todo desde un único binario ligero y extremadamente rápido.

---

# ✨ Características Principales

## ⚡ Rendimiento Extremo
- Carga progresiva inteligente
- Caché optimizado
- Paralelización avanzada
- Actualizaciones en tiempo real mediante SSE

## ☸️ Kubernetes Visualization
- Topologías interactivas
- Relaciones entre servicios
- Flujo de tráfico en vivo
- Monitoreo visual de workloads

## 📦 Helm Management
- Gestión de releases
- Rollbacks
- Historial de despliegues
- Actualizaciones desde UI

## 🔄 GitOps Support
- Compatible con:
  - FluxCD
  - ArgoCD
- Estado de sincronización
- Reconciliación automática

## 🤖 Integración con IA (MCP)
- Compatible con:
  - Claude
  - Cursor
  - Copilot
- Datos optimizados para LLMs
- Herramientas AI-ready integradas

## 🔐 Seguridad
- RBAC
- OIDC Authentication
- Proxy Authentication
- Soporte Air-Gapped

## 📊 Cluster Monitoring
- Timeline de eventos
- Logs en tiempo real
- Resource Browser
- Auditorías del cluster

---

# 🏗️ Arquitectura

## Backend
- Go 1.26+
- Kubernetes API
- SSE
- SQLite
- Prometheus/OpenCost

## Frontend
- SPA moderna
- Visualización interactiva
- ELK.js
- UI responsiva

---

# 🧰 Tecnologías Utilizadas

| Tecnología | Uso |
|---|---|
| Go | Backend principal |
| Kubernetes API | Gestión del cluster |
| Helm | Gestión de paquetes |
| FluxCD | GitOps |
| ArgoCD | GitOps |
| Prometheus | Métricas |
| OpenCost | Costos |
| SQLite | Timeline storage |
| SSE | Actualizaciones realtime |
| ELK.js | Layout de topologías |

---

# 🚀 Instalación

## Instalación rápida

```bash
curl -fsSL https://get.radarhq.io | sh
```

## Ejecutar Radar

```bash
kubectl radar
```

o simplemente:

```bash
radar
```

---

# 🍺 Instalación con Homebrew

```bash
brew install skyhook-io/tap/radar
```

---

# 🪟 Instalación en Windows

```powershell
irm https://get.radarhq.io/install.ps1 | iex
```

---

# ☸️ Instalación In-Cluster

```bash
helm repo add skyhook https://skyhook-io.github.io/helm-charts

helm install radar skyhook/radar \
  -n radar \
  --create-namespace
```

---

# 📊 Funcionalidades

## 🗺️ Topology View
Visualiza cómo se conectan los recursos Kubernetes en tiempo real.

- Relaciones entre servicios
- Tráfico entre pods
- Agrupación por namespace
- Auto-layout dinámico

---

## 📦 Resource Browser
Explora todos los recursos del cluster.

- Deployments
- Pods
- StatefulSets
- CRDs
- Services
- Nodes

---

## 🧾 Timeline
Monitorea eventos y cambios del cluster.

- Eventos en tiempo real
- Warnings
- Diffs de recursos
- Historial de cambios

---

## 🚦 Traffic Visualization
Monitoreo de tráfico usando:

- Hubble
- Caretta
- Istio

---

## 🔍 Image Filesystem Viewer
Explora el filesystem interno de imágenes Docker directamente desde Kubernetes.

---

## 💰 Cost Insights
Análisis de costos usando OpenCost.

- Costos por namespace
- Costos por workload
- Eficiencia de recursos

---

## 🔐 TLS Certificate Management
Monitorea certificados TLS y fechas de expiración.

---

## 🛡️ Cluster Audit
Auditorías automáticas del cluster:

- Seguridad
- Buenas prácticas
- Performance
- Reliability

---

# 🤖 MCP AI Integration

Radar incluye un servidor MCP integrado para asistentes IA.

Compatible con:

- Claude
- Cursor
- GitHub Copilot
- Otros asistentes MCP

Permite consultar:

- Eventos
- Topologías
- Logs
- Estado del cluster
- Auditorías

Sin consumir enormes cantidades de contexto en los LLMs.

---

# 📁 Recursos Soportados

## Kubernetes Core
- Pods
- Deployments
- StatefulSets
- Services
- Ingress
- ConfigMaps
- Secrets
- PVCs

## GitOps
- FluxCD
- ArgoCD

## Seguridad
- Kyverno
- Trivy
- cert-manager

## Networking
- Istio
- Gateway API
- Traefik
- Contour

## Escalado
- KEDA
- HPA
- VPA
- Karpenter

## Otros
- Velero
- External Secrets
- Knative
- Prometheus Operator
- OpenCost

---

# ⌨️ Shortcuts

| Shortcut | Acción |
|---|---|
| `1-6` | Cambiar vista |
| `t` | Cambiar tema |
| `?` | Ver shortcuts |
| `⌘K` | Command palette |
| `/` | Buscar |
| `f` | Ajustar topología |
| `+/-` | Zoom |

---

# 🛠️ Desarrollo

## Clonar proyecto

```bash
git clone https://github.com/skyhook-io/radar.git
```

## Entrar al proyecto

```bash
cd radar
```

## Instalar dependencias

```bash
make deps
```

## Frontend

```bash
make watch-frontend
```

## Backend

```bash
make watch-backend
```

---

# 📌 Requisitos

- Kubernetes Cluster
- kubectl
- Go 1.26+
- Helm (opcional)

---

# 🌎 Sitio Oficial

🔗 https://radarhq.io

---

# 📚 Documentación

📖 https://radarhq.io/docs

---

# 🧠 Casos de Uso

- DevOps
- SRE
- Kubernetes Administration
- Platform Engineering
- GitOps Monitoring
- Cluster Auditing
- Cloud Native Monitoring

---

# 👨‍💻 Desarrollador

## Isai Reyes

Desarrollador Full Stack especializado en:

- Kubernetes
- DevOps
- Cloud Native
- Spring Boot
- React
- Vue
- Docker
- Ciberseguridad

---

# ⭐ Support

Si te gusta este proyecto:

- Dale ⭐ en GitHub
- Haz Fork 🍴
- Comparte el proyecto 🚀

---

# 📜 Licencia

Este proyecto está bajo la licencia Apache 2.0.

---

<div align="center">

### ☸️ Radar — Kubernetes Visibility Platform

Desarrollado y personalizado por **Isai Reyes** 🚀

</div>
