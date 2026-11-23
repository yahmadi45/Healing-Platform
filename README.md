# SentinelOps — PFE MVP

**Projet 10 — DevOps, SSO et supervision**

SentinelOps est un MVP pédagogique d'une plateforme interne d'opérations pour services conteneurisés. Cette première version fournit une console web, une API Go, des services de démonstration, une simulation d'incident/récupération, des métriques Prometheus et une configuration Docker Compose.

> **Limite importante :** le MVP ne se connecte pas au daemon Docker et ne redémarre aucun conteneur. Les boutons « Simulate failure » et « Recover » modifient uniquement l'état en mémoire de l'application. Ils servent à démontrer un flux de détection/récupération sans risque. Il n'y a pas encore de Keycloak/SSO, base persistante, agent Linux, moteur IA, NATS, déploiement réel, sauvegarde ni rollback.

## Sommaire

- [Objectifs](#objectifs)
- [Architecture livrée](#architecture-livrée)
- [Structure du dépôt](#structure-du-dépôt)
- [Prérequis](#prérequis)
- [Lancer avec Docker Compose](#lancer-avec-docker-compose)
- [Utiliser la console](#utiliser-la-console)
- [API HTTP](#api-http)
- [Métriques](#métriques)
- [Tests et développement local](#tests-et-développement-local)
- [Configuration](#configuration)
- [Sécurité et limites](#sécurité-et-limites)
- [Architecture cible du PFE](#architecture-cible-du-pfe)
- [Plan de réalisation](#plan-de-réalisation)
- [Répartition entre deux PC](#répartition-entre-deux-pc)
- [Dépannage](#dépannage)

## Objectifs

Ce dépôt donne une base exécutable pour commencer à explorer les axes du sujet :

1. Conteneuriser une application et ses services.
2. exposer un tableau de bord d'état et un historique d'incidents.
3. mesurer les états avec Prometheus.
4. expliquer un flux fermé d'incident : détection → politique → action contrôlée → vérification.
5. construire ensuite les fonctions DevSecOps, SSO, supervision et reprise après sinistre comme livrables séparés.

Ne présentez pas les extensions prévues comme des fonctions déjà implémentées.

## Architecture livrée

```text
Navigateur
   │ HTTP :8080
   ▼
SentinelOps (Go, API + fichiers statiques embarqués)
   ├── /api/services       États de démo en mémoire
   ├── /api/incidents      Incidents de démo en mémoire
   ├── /api/events         Journal d'événements de démo
   ├── /metrics            Exposition Prometheus
   └── /healthz, /readyz   Vérifications de santé
          ▲
          │ scrape toutes les 15 secondes
          └──────── Prometheus :9090
```

- **Backend :** Go standard library, HTTP, stockage volatile protégé par mutex.
- **UI :** HTML, CSS et JavaScript natifs, servis par le même binaire Go. Aucune compilation frontend ni installation npm nécessaire.
- **Image :** compilation multi-stage; l'image d'exécution utilise Alpine et un utilisateur non-root.
- **Prometheus :** configuration locale `prometheus.yml`, rétention de démonstration de 7 jours. Aucun volume durable n'est configuré pour Prometheus dans ce MVP.
- **Healthcheck :** l'image vérifie `/healthz`.
- **Actions :** les scénarios sont simulés, ne contrôlent pas les conteneurs.

## Structure du dépôt

```text
.
├── .dockerignore
├── Dockerfile
├── README.md
├── docker-compose.yml
├── go.mod
├── main.go
├── main_test.go
├── prometheus.yml
└── web/
    ├── app.js
    ├── index.html
    └── style.css
```

## Prérequis

- Docker Engine avec le plugin Compose, ou Docker Desktop.
- Git si vous clonez le dépôt.
- Ports locaux libres : `8080` pour SentinelOps et `9090` pour Prometheus.

Aucun compte externe, clé API, modèle IA ou service cloud n'est requis pour lancer le MVP.

## Lancer avec Docker Compose

Depuis le répertoire du projet :

```sh
docker compose up --build --pull always
```

`--pull always` vérifie les images de service configurées avant le démarrage; `--build` reconstruit l'image SentinelOps. Ouvrez ensuite :

- Console : <http://localhost:8080>
- Prometheus : <http://localhost:9090>
- Santé API : <http://localhost:8080/healthz>
- Métriques : <http://localhost:8080/metrics>

Arrêter sans supprimer les données de Docker :

```sh
docker compose down
```

Reconstruire uniquement l'application après une modification :

```sh
docker compose up --build --pull always -d sentinelops
```

Les données des incidents sont conservées uniquement tant que le processus Go reste en cours. Elles disparaissent lors d'un redémarrage du conteneur.

## Utiliser la console

1. La page affiche les services `api`, `web` et `postgres` dans l'état initial `healthy`.
2. Cliquer sur **Simulate failure** marque le service comme `unhealthy` dans l'état en mémoire et crée un incident de démonstration.
3. Cliquer sur **Recover** marque le service comme `healthy`, résout les incidents ouverts correspondants et incrémente son compteur de récupérations.
4. Le tableau de bord actualise les données toutes les 15 secondes. Le bouton **Refresh data** permet un rafraîchissement immédiat.
5. Un service est limité à trois récupérations simulées. C'est une limite de démonstration, pas un circuit breaker distribué.

Les intitulés indiquent explicitement que les changements sont simulés; aucun processus externe n'est arrêté.

## API HTTP

Toutes les réponses API sont au format JSON sauf `/metrics`.

| Méthode | Route | Fonction |
|---|---|---|
| `GET` | `/api/services` | Liste des services simulés |
| `GET` | `/api/incidents` | Historique d'incidents simulés |
| `GET` | `/api/events` | Événements de simulation |
| `GET` | `/api/stats` | Résumé des compteurs |
| `POST` | `/api/services/{name}/fail` | Simule une panne et crée un incident |
| `POST` | `/api/services/{name}/recover` | Simule le rétablissement et résout l'incident |
| `GET` | `/healthz` | Santé du processus |
| `GET` | `/readyz` | Disponibilité de démonstration |
| `GET` | `/metrics` | Métriques au format Prometheus |

Exemple de parcours :

```sh
curl http://localhost:8080/api/services
curl -X POST http://localhost:8080/api/services/api/fail
curl http://localhost:8080/api/incidents
curl -X POST http://localhost:8080/api/services/api/recover
curl http://localhost:8080/api/stats
```

Une requête de récupération vers un service sain renvoie `409 Conflict`; un nom inconnu renvoie `404`; une action désactivée par configuration renvoie `403`.

## Métriques

Les métriques actuellement exposées sont :

- `sentinelops_services_total` — nombre de services de démonstration.
- `sentinelops_services_unhealthy` — services de démonstration non sains.
- `sentinelops_incidents_open` — incidents ouverts.
- `sentinelops_recoveries_total` — récupérations simulées réussies.

Prometheus scrape SentinelOps toutes les 15 secondes. Vérifiez les cibles sur <http://localhost:9090/targets>. La métrique représente l'état du simulateur, pas la santé réelle du daemon Docker, d'un hôte Linux ou d'une base de données.

## Tests et développement local

Lancer les tests unitaires :

```sh
go test ./...
```

Vérifier le format Go :

```sh
gofmt -w main.go main_test.go
go vet ./...
```

Lancer directement le serveur :

```sh
go run .
```

Puis visiter <http://localhost:8080>. Les fichiers web sont intégrés au binaire avec `go:embed`.

## Configuration

Les variables lues par le service :

| Variable | Défaut | Description |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | Adresse/port d'écoute interne du serveur |
| `ALLOW_DEMO_ACTIONS` | `true` | Autorise les changements d'état simulés |
| `SENTINELOPS_PORT` | `8080` | Port hôte publié par Compose |
| `PROMETHEUS_PORT` | `9090` | Port hôte publié par Compose |

Pour désactiver les boutons d'action :

```powershell
$env:ALLOW_DEMO_ACTIONS="false"; docker compose up --build --pull always
```

Ou en bash :

```sh
ALLOW_DEMO_ACTIONS=false docker compose up --build --pull always
```

Compose utilise également un `.env` local pour substituer les ports. Ne mettez jamais secrets, mots de passe ou jetons dans une image, dans Git ou dans ce fichier de configuration public. Le MVP ne nécessite pas de secrets.

## Sécurité et limites

Cette application est une base pédagogique, **pas un plan de contrôle prêt pour la production**.

- Il n'y a pas d'authentification, SSO, RBAC, CSRF protection ou séparation des rôles. Ne l'exposez pas à Internet ni à un réseau non fiable.
- Les actions sont actuellement simulées et non authentifiées. Gardez le service sur votre machine ou dans un laboratoire isolé.
- Pour un accès depuis le PC léger, préférez un tunnel SSH local plutôt que publier le port sur un réseau public. Exemple depuis le PC client : `ssh -L 8080:127.0.0.1:8080 utilisateur@IP_DU_PC_HOTE`, puis visitez `http://localhost:8080`. Adaptez le nom d'utilisateur et l'adresse IP.
- Avant toute connexion future au daemon Docker, créer une interface minimale de commandes autorisées, authentifier chaque demande, appliquer least privilege, auditer et limiter les actions. Ne jamais accepter une commande shell arbitraire envoyée par HTTP.
- Les états et incidents sont volatils; ils ne peuvent servir comme audit durable.
- Prometheus n'a pas de volume nommé dans cet exemple; ses données sont éphémères selon le cycle de vie du conteneur.
- Les polices distantes sont décoratives; l'interface fonctionne sans elles, mais le chargement distant peut ne pas fonctionner hors ligne.

## Architecture cible du PFE

Élargir progressivement, sans commencer par neuf microservices. Chaque étape doit être testable et correspondre à un besoin mesurable.

### Étape 1 — socle fiable

- Définir les modèles de services, déploiements et incidents.
- Remplacer le stockage mémoire par PostgreSQL et migrations versionnées.
- Ajouter tests d'intégration, gestion de configuration et journaux structurés.
- Définir les contrats d'API et l'idempotency key des actions.

### Étape 2 — identité et audit

- Intégrer Keycloak via OpenID Connect.
- Valider l'émetteur, l'audience, la signature et l'expiration des jetons.
- Mettre en place des rôles Viewer, Operator et Admin avec autorisation serveur.
- Journaliser acteur, décision, cible, résultat, horodatage et correlation ID.
- Ajouter tests qui prouvent qu'un Viewer ne peut pas déclencher une action.

### Étape 3 — déploiements contrôlés

- Construire des images dans CI et leur attribuer un tag immuable (idéalement digest).
- Enregistrer l'historique de versions et lancer un déploiement vers un environnement de test.
- Vérifier health checks; n'autoriser le rollback que selon une politique explicite.
- Ajouter approbation humaine avant actions à risque.
- Utiliser des cibles de démonstration dédiées, jamais les services personnels ou production.

### Étape 4 — observabilité et incidents

- Ajouter Prometheus, Alertmanager, Grafana, Loki et OpenTelemetry selon les besoins.
- Transmettre un correlation ID entre API, déploiement et événements.
- Dédupliquer les alertes avec une clé stable et un intervalle de regroupement.
- Construire une timeline qui relie déploiements, alertes, logs et décisions.
- Distinguer clairement alertes observées, causes suspectées et causes confirmées.

### Étape 5 — récupération sûre

- Modéliser les actions autorisées avec schémas fermés; aucune commande shell libre.
- Ajouter idempotence, limite de répétition, cooldown, timeout et verrou par service.
- Faire d'abord le mode dry-run puis exiger approbation humaine.
- Vérifier la santé après action et enregistrer succès/échec.
- Limiter les permissions système à une petite allowlist de services isolés.

### Étape 6 — sécurité de livraison

- Pipeline CI : format, tests, lint, build, scans dépendances et image, secrets, SBOM.
- Définir des seuils de blocage justifiés; documenter les exceptions et leur expiration.
- Tester les configurations Docker, utilisateur non-root, ports, capabilities et secrets.
- Conserver les artefacts et résultats de scan liés à une révision de code.

### Étape 7 — sauvegarde et reprise

- Définir RPO/RTO avec l'encadrant et le scénario d'essai.
- Sauvegarder PostgreSQL selon une politique de rétention documentée.
- Protéger les clés de chiffrement séparément des sauvegardes.
- Tester régulièrement restauration dans un environnement isolé.
- Mesurer les résultats de restauration; une archive non restaurée n'est pas une preuve de reprise.

### Étape 8 — système multi-hôte (extension)

- Concevoir un agent Go à protocole authentifié, identité par hôte et permissions minimales.
- Gérer heartbeats, pertes réseau, reprise de connexion et commandes idempotentes.
- Ajouter NATS seulement si le besoin d'événements asynchrones justifie son coût opérationnel.
- Simuler partition réseau, arrêt du control plane, panne base, saturation disque et rollback.
- Mesurer latence de détection, taux de faux positifs, durée de récupération et RPO/RTO.

### Composants: livré vs extension

| Composant | État |
|---|---|
| Go API + interface web | Livré |
| Dockerfile multi-stage + Compose | Livré |
| Prometheus scrape + endpoint métriques | Livré |
| Incidents/récupération de démonstration | Livré, mémoire seulement |
| PostgreSQL durable | À développer |
| Keycloak SSO / OIDC / RBAC | À développer |
| Grafana / Loki / traces | À intégrer/configurer |
| CI/CD et scanners DevSecOps | À configurer |
| Docker host agent / multi-host | Extension, non implémentée |
| NATS / event streaming | Extension, non implémentée |
| Backup/restore testés | À développer |
| IA / root-cause analysis automatique | Non incluse; ne pas promettre sans évaluation et mesures |

## Répartition entre deux PC

Si le premier PC possède beaucoup de stockage et le second environ 225 GB :

- **PC hôte (stockage élevé) :** dépôt de travail, Docker Engine, images, volumes, PostgreSQL de labo, Prometheus et build cache.
- **PC client (225 GB) :** éditeur (VS Code), Git, navigateur et connexion SSH. Faire du Remote SSH vers le PC hôte permet de garder outils et `node_modules` éventuels du côté hôte; ce dépôt n'a actuellement aucun `node_modules`.
- **Git :** chaque personne ou agent travaille sur une branche ou un worktree séparé; ouvrir une Pull Request et faire tourner les tests avant merge.
- **Agents IA :** découper en tâches indépendantes (ex. tests API, UI, configuration observabilité). Revoir les changements et ne pas laisser plusieurs agents modifier simultanément le même répertoire de travail.
- **Données :** sauvegarder séparément les volumes avant une opération de nettoyage; les images Docker sont retéléchargeables mais leurs volumes peuvent contenir des données irremplaçables.

Pour le tunnel SSH, le PC hôte doit avoir un serveur SSH accessible depuis le PC client. N'ouvrez pas le port Docker daemon 2375 sur le LAN; l'API Docker donne des pouvoirs élevés sur la machine.

## Dépannage

### Port déjà occupé

```powershell
$env:SENTINELOPS_PORT="8081"; docker compose up --build --pull always
```

Interface : `http://localhost:8081`.

### Voir logs et état des services

```sh
docker compose ps
docker compose logs -f sentinelops
docker compose logs -f prometheus
```

### Reconstruire sans cache

```sh
docker compose build --no-cache sentinelops
docker compose up --pull always
```

### Arrêt complet

```sh
docker compose down
```

La commande ci-dessus ne supprime pas les autres volumes Docker de la machine.
