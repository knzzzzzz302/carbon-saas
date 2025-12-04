CarbonV2 - Monorepo
====================

Ce dépôt contiendra l’implémentation du SaaS CarbonV2 (inspiré de Greenly) et **s’inspire explicitement du dépôt GitHub suivant : https://github.com/incubateur-ademe/nosgestesclimat**.  
Les modules, conventions et idées de modèle de calcul GES (dont l’usage de Publicodes) sont utilisés comme point de départ – toute adaptation doit être tracée dans l’historique Git.

Structure cible (MVP à V2) :

- `apps/marketing` : site marketing public (Next.js)
- `apps/frontend` : application SaaS (Next.js)
- `services/api` : backend Go principal (API, jobs)
- `services/ml` : service ML (Mistral, classification, recommandations)
- `infra` : IaC, CI/CD, scripts de déploiement

Pour l’instant, seul le backend Go minimal sera mis en place, avec une exécution possible via :

```bash
cd services/api
go run main.go
```

Une documentation détaillée sera ajoutée au fur et à mesure (architecture, API, déploiement).


