---
title: 'Construire AgentsMesh avec AgentsMesh : 52 jours de Harness Engineering en solo'
excerpt: "OpenAI a inventé le terme Harness Engineering. J'ai appliqué cette méthodologie seul pendant 52 jours — 600 commits, 965 687 lignes de code traitées — pour construire l'outil de Harness Engineering lui-même. Le codebase comme contexte, l'environnement d'ingénierie comme plafond de qualité des agents."
date: "2026-03-04"
author: "AgentsMesh Team"
category: "Insight"
readTime: 12
---

OpenAI a récemment publié un article décrivant comment ils ont utilisé des agents IA pour produire plus d'un million de lignes de code en cinq mois. Ils ont baptisé cette discipline **Harness Engineering**.

J'ai commencé à construire **AgentsMesh** il y a un peu plus de 50 jours. 52 jours, 600 commits, 965 687 lignes de code traitées, 356 220 lignes de code en production. Une seule personne.

Mais ce qui mérite d'être raconté, ce ne sont pas les chiffres. C'est la structure même de l'entreprise : j'ai utilisé le Harness Engineering pour construire un outil de Harness Engineering.

Le dépôt est entièrement open source, l'historique Git est public. Tous les chiffres sont vérifiables via git log.

## L'environnement d'ingénierie détermine le plafond de qualité des agents

52 jours de travail concret m'ont convaincu d'une chose : la qualité de la production d'un agent ne dépend pas uniquement de l'agent lui-même, mais surtout du terreau d'ingénierie dans lequel il opère. Voici les choix qui se sont réellement sédimentés dans le codebase.

### Une architecture en couches pour que l'agent sache où intervenir

Le codebase suit un découpage **DDD** strict : la couche domain ne contient que les structures de données, la couche service uniquement la logique métier, la couche handler uniquement la transformation des formats HTTP. 22 modules de domaine, des frontières nettes, un fichier interface.go par module qui définit explicitement le contrat public.

Quand un agent doit ajouter une fonctionnalité, il sait : les structures de données vont dans domain, les règles métier dans service, les routes dans handler. Un codebase aux frontières floues pousse l'agent à placer le code au mauvais endroit ; un codebase aux frontières claires produit du code qui s'intègre naturellement. Ce n'est pas de la propreté architecturale théorique — c'est la carte de navigation de l'agent au moment de générer du code.

### La structure des répertoires comme documentation

Le nommage est aligné de bout en bout, du backend au frontend. Prenons Loop comme exemple : backend/internal/domain/loop/ pour les structures de données, backend/internal/service/loop/ pour la logique métier, web/src/components/loops/ pour les composants frontend. Le concept produit se projette directement sur le chemin du code, sans recherche nécessaire — le nom du répertoire est la carte.

Les 16 modules domain du backend (agentpod, channel, ticket, loop, runner...) sont en miroir 1:1 avec la couche service ; les components du frontend sont organisés par fonction produit (pod, tickets, loops, mesh, workspace) et alignés sur le nommage du backend. Un agent qui reçoit une tâche liée aux Tickets n'a pas besoin d'explorer l'ensemble du codebase — l'arborescence lui indique directement où intervenir.

Cette convention n'a jamais été écrite dans un document. Elle a été renforcée en continu par chaque commit d'agent dans le codebase.

### La dette technique est amplifiée exponentiellement par les agents

C'est l'une des découvertes les plus contre-intuitives de ces 52 jours.

Quand vous faites un compromis temporaire dans un module — contourner la couche service pour accéder directement à la base de données, ou utiliser une valeur en dur — l'agent apprend ce pattern. La prochaine fois qu'il génère une fonctionnalité similaire, il réutilise ce « précédent ». Pas de manière ponctuelle, mais de manière systémique. La dette technique ne reste plus isolée : elle se propage.

Un ingénieur humain qui tombe sur du mauvais code sait généralement que « c'est un piège, il faut le contourner ». L'agent ne porte pas ce jugement — il constate : ce pattern existe dans le codebase, donc c'est une pratique légitime.

Cela signifie que le signal de qualité du code dans le dépôt est bien plus critique qu'en développement traditionnel. Si les bonnes pratiques dominent, l'agent les amplifie ; si les compromis temporaires dominent, l'agent amplifie la dette technique.

Concrètement, j'ai dû m'arrêter plusieurs fois pour nettoyer exclusivement la dette technique — pas de nouvelles fonctionnalités, uniquement du refactoring. Non pas pour rendre le code « joli », mais pour maintenir la pureté du signal d'ingénierie dans le dépôt. C'est un coût de maintenance propre au développement assisté par agent, et l'une des différences majeures avec le rythme de développement traditionnel.

### Le typage fort comme barrière de qualité à la compilation

Go + TypeScript + Proto. Le typage fort fait remonter une masse d'erreurs du runtime vers la compilation.

L'agent génère une fonction dont la signature ne correspond pas ? Échec de compilation. L'agent modifie un format d'API sans mettre à jour la définition de type ? TypeScript signale l'erreur immédiatement. L'agent change le format de message du Runner sans synchroniser le Backend ? Le code généré par Proto ne compile pas.

Dans un langage faiblement typé, ces erreurs se glisseraient silencieusement en runtime. Le typage fort les bloque avant le commit. Plus la boucle de feedback est courte, plus l'agent itère efficacement.

### Quatre couches de boucles de feedback

Un agent a besoin de savoir rapidement ce qu'il a fait de travers. Une seule couche ne suffit pas, quatre est le bon équilibre. Plus la boucle de feedback est courte et précise, meilleur est le résultat livré par l'agent.

Première couche : la compilation. Hot reload via Air, redémarrage du code Go en moins d'une seconde ; les erreurs de type TypeScript sont signalées en temps réel. Les erreurs de syntaxe et de typage sont éliminées à ce niveau.

Deuxième couche : les tests unitaires. Plus de 700 tests couvrent les couches domain et service. L'agent sait en moins de 5 minutes s'il a introduit une régression, notamment sur des conditions limites comme l'isolation multi-tenant, que les agents ont tendance à négliger.

Troisième couche : les tests e2e. Validation de bout en bout des parcours fonctionnels réels. Ils couvrent les frontières d'intégration que les tests unitaires ne peuvent pas atteindre — l'interaction réelle entre plusieurs modules.

Quatrième couche : le pipeline CI. Chaque PR déclenche automatiquement la suite complète de tests, le linting, la vérification de types et la validation de build multi-plateformes. Le dernier filet de sécurité avant le merge, exécuté par la machine, indépendant de la vigilance du relecteur.

La latence augmente d'une couche à l'autre, tout comme l'éventail d'erreurs détectées. L'agent modifie une ligne ? La première couche valide. L'agent effectue un refactoring inter-modules ? Seule la quatrième couche peut valider complètement.

### Le worktree pour le parallélisme natif

dev.sh calcule automatiquement un décalage de ports en fonction du nom du worktree Git, attribuant un intervalle de ports indépendant à chaque worktree. Plusieurs agents travaillent simultanément sur différents worktrees, avec des environnements totalement isolés, sans gestion manuelle des conflits de ports.

C'est l'extension de la primitive d'isolation Pod au niveau de l'environnement de développement — la même logique, portée de l'environnement d'exécution des agents à leur environnement de développement.

### Le codebase comme contexte de l'agent, pas seulement un prompt

En assemblant tous ces éléments, on arrive à une conclusion unique : le codebase lui-même est le contexte le plus important dans lequel l'agent travaille. L'architecture en couches indique à l'agent où modifier ; la structure des répertoires indique quel fichier chercher ; le niveau de propreté de la dette technique détermine si l'agent apprend de bons ou de mauvais patterns ; la densité de tests détermine l'audace avec laquelle l'agent peut refactorer ; le typage fort détermine la précocité de détection des erreurs.

Cela signifie qu'il n'est pas nécessaire de construire un système de contexte externe au codebase — pas besoin de faire du Context Engineering dédié, pas besoin de monter un RAG séparé, pas besoin de maintenir des fichiers de mémoire supplémentaires. Ce qu'il faut, c'est faire du codebase lui-même un contexte de haute qualité. **Le dépôt est le contexte.**

**C'est aussi pourquoi l'investissement en Harness Engineering converge avec le génie logiciel classique** : écrire du code clair, maintenir une bonne architecture, nettoyer la dette technique régulièrement. La seule différence est l'objectif — avant, c'était pour faciliter la maintenance par des ingénieurs humains ; désormais, c'est aussi pour permettre aux agents IA de travailler de manière fiable.

## La bande passante cognitive est une contrainte d'ingénierie réelle

Vers le cinquième jour, j'ai heurté un mur bien réel : environ 50 000 lignes de débit quotidien.

Trois worktrees ouverts simultanément, trois agents en cours d'exécution, et moi qui naviguais entre eux pour prendre des décisions. Ajouter un quatrième agent faisait chuter visiblement la qualité des décisions. Ce n'était pas une impression — j'ai découvert a posteriori que cette période avait produit plusieurs décisions architecturales médiocres.

Le débit quotidien de 50 000 lignes n'est pas une limite d'outillage, c'est le plafond naturel de la bande passante cognitive humaine. On peut prendre des décisions architecturales véritables pour environ trois flux de travail parallèles ; au-delà, la qualité commence à se dégrader.

La seule façon de dépasser ce plafond : échanger la délégation contre l'échelle. Non pas donner plus de tâches aux agents, mais déléguer la prise de décision elle-même. Laisser les agents coordonner les agents, et monter d'un niveau — passer de la supervision d'un agent individuel à la supervision du système qui supervise les agents. C'est ainsi qu'est né le mode **Autopilot**.

C'est l'intention de conception fondamentale d'AgentsMesh. Et c'est quelque chose que je n'ai vraiment compris qu'en l'utilisant pour se construire lui-même.

## L'effondrement du coût de l'erreur transforme la méthodologie d'ingénierie

L'architecture Relay d'AgentsMesh n'a pas été conçue sur un tableau blanc. Elle a été forgée en production.

Trois Pods en exécution simultanée ont fait tomber le Backend. J'ai observé le crash, compris la cause, et reconstruit. Ajout du Relay pour isoler le trafic des terminaux. Nouveaux problèmes, ajout de l'agrégation intelligente, ajout de la gestion de connexions à la demande. L'architecture finale est née d'une succession de pannes réelles, pas d'une discussion théorique.

L'intuition d'ingénierie traditionnelle dit : concevoir d'abord, construire ensuite — anticiper exhaustivement les cas limites, car le coût de l'erreur est élevé.

Quand le coût de l'erreur tend vers zéro, cette intuition devient un handicap.

Cette panne du Relay a été résolue en moins de deux jours, de la détection au correctif. Dans une équipe traditionnelle, c'est deux semaines de discussions architecturales — et la discussion aurait inévitablement oublié quelque chose.

**L'IA ne change pas la vitesse d'écriture du code. Elle change la structure de coûts de l'ensemble du processus d'ingénierie.** Quand l'itération est suffisamment bon marché, l'approche expérimentale produit de meilleures architectures que l'approche par conception — et plus vite. Le critère de justesse d'une architecture n'est plus de passer une revue, mais de survivre à la production.

## Validation par auto-amorçage

La thèse centrale d'AgentsMesh : les agents IA peuvent, dans un Harness structuré, collaborer pour mener à bien des tâches d'ingénierie complexes.

J'ai utilisé AgentsMesh pour construire AgentsMesh.

C'est le test le plus direct de cette thèse. Si le Harness Engineering fonctionne réellement, l'outil doit être capable de se construire lui-même.

52 jours, 965 687 lignes de code traitées, 356 220 lignes de code en production, 600 commits, un seul auteur.

OpenAI était une équipe, sur cinq mois. Ce n'est pas une comparaison — les contextes sont différents, les échelles aussi. Mais une chose est identique : le Harness rend possible une production qui serait autrement inaccessible.

L'historique des commits est la preuve. N'importe quel ingénieur peut cloner le dépôt, exécuter git log --numstat — les chiffres ne changent pas selon qui les consulte.

## Trois primitives d'ingénierie

52 jours de pratique et de validation par auto-amorçage ont fait converger le travail vers trois primitives d'ingénierie. Ce n'est pas un cadre produit conçu à l'avance — ce sont des réponses forgées par des problèmes d'ingénierie réels.

**Isolation**
Chaque agent a besoin de son propre espace de travail indépendant. Ce n'est pas une bonne pratique, c'est un prérequis absolu. Sans isolation, le travail en parallèle est structurellement impossible. AgentsMesh implémente cela avec le **Pod** : chaque agent s'exécute dans son propre worktree Git et son propre sandbox. Les conflits passent de « possibles » à « structurellement impossibles ». L'isolation implique aussi la cohésion : dans l'environnement isolé du Pod, tout le contexte nécessaire à l'exécution de l'agent est préparé — Repo, Skills, MCP et plus encore. Construire un Pod, c'est préparer l'environnement d'exécution de l'agent.

**Décomposition**
Les agents ne sont pas efficaces face à « arrange-moi ce codebase ». Ils excellent quand on leur dit : tu possèdes ce périmètre, voici les critères d'acceptation, voici la définition de terminé. L'appropriation ne se réduit pas à l'attribution de tâches — elle transforme la façon dont l'agent raisonne. La décomposition est le travail d'ingénierie qui doit être accompli avant toute exécution d'agent.

AgentsMesh propose deux abstractions pour la décomposition : le **Ticket** correspond à un travail ponctuel — développement de fonctionnalité, correction de bug, refactoring, avec un flux kanban complet et le suivi des Merge Requests ; le **Loop** correspond à une tâche automatisée récurrente — tests quotidiens, builds planifiés, audits de qualité du code, pilotés par des expressions Cron, chaque exécution laissant un LoopRun indépendant. Deux formes de tâches aux frontières claires : faire une chose, utiliser un Ticket ; faire la même chose régulièrement, utiliser un Loop.

**Coordination**
Nous n'avons pas utilisé l'abstraction de rôles pour organiser la collaboration entre agents. Les équipes traditionnelles ont besoin de rôles parce que chaque personne ne maîtrise que quelques spécialités — un développeur frontend n'écrit pas le backend, un product manager n'écrit pas de code. Mais les agents ne sont pas soumis à cette contrainte : un même agent peut écrire du code, générer de la documentation, faire de l'analyse concurrentielle, exécuter des tests, relire des PRs, et même orchestrer les workflows d'autres agents. Ses capacités ne sont pas figées — elles sont configurées par le contexte et les outils. La collaboration entre agents n'a donc pas besoin de reproduire la division du travail humaine. Elle a besoin de communication et de permissions.

Le **Channel** résout la dimension collective : plusieurs Pods partagent messages, décisions et documents dans un espace de collaboration commun. C'est le fondement qui permet à un agent Supervisor et à des agents Worker de former une structure de collaboration — pas un chat de groupe, mais une couche de communication structurée avec historique contextualisé.

Le **Binding** résout la dimension des capacités : une autorisation point à point entre deux Pods. **pod:read** permet à un agent d'observer la sortie terminal d'un autre agent ; **pod:write** permet à un agent de contrôler directement l'exécution d'un autre. Le Binding est le mécanisme par lequel un agent coordonne un autre agent — le Supervisor ne perçoit pas l'état du Worker en envoyant des messages, mais en regardant directement son terminal.

OpenAI appelle les concepts équivalents Context Engineering, contraintes architecturales et gestion de l'entropie. Les noms diffèrent, le problème résolu est le même.

## Open source

Le Harness Engineering est une discipline d'ingénierie, pas une fonctionnalité produit. Plutôt que de la garder pour nous, nous préférons la partager pour inspirer.

Nous avons choisi de rendre AgentsMesh open source. Quand nous construisons ce qui pourrait être un outil d'ingénierie efficace, l'objectif n'a jamais été de « posséder le code », mais de permettre à d'autres de construire de meilleurs outils d'ingénierie sur cette base. Plutôt que d'enfermer des pratiques potentiellement justes dans un produit propriétaire, nous les ouvrons pour que la communauté les valide, les fasse évoluer et les dépasse.

Le code est sur [GitHub](https://github.com/AgentsMesh/AgentsMesh)

Vous pouvez l'utiliser pour : déployer votre propre Runner et exécuter des agents IA dans des environnements isolés locaux ; gérer les workflows d'agents avec Ticket et Loop ; faire collaborer plusieurs agents sur des tâches complexes via Channel et Binding.

Si vous faites vos propres découvertes en pratiquant le Harness Engineering, venez en discuter sur [GitHub Discussions](https://github.com/AgentsMesh/AgentsMesh/discussions) ou ouvrez directement une [Issue](https://github.com/AgentsMesh/AgentsMesh/issues). Ce projet a été construit par des agents — il est naturel qu'il continue d'évoluer grâce aux agents et aux ingénieurs ensemble.
