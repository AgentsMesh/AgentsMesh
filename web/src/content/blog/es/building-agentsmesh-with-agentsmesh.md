---
title: 'Construir AgentsMesh con AgentsMesh: 52 dias de Harness Engineering en solitario'
excerpt: "OpenAI lo llama Harness Engineering. Una sola persona, 52 dias, 600 commits, 965,687 lineas de codigo procesadas y 356,220 lineas en produccion. Asi se construye una herramienta de Harness Engineering usando Harness Engineering."
date: "2026-03-04"
author: "AgentsMesh Team"
category: "Insight"
readTime: 12
---

OpenAI publico recientemente un articulo describiendo como utilizaron agentes de IA para producir mas de un millon de lineas de codigo en cinco meses. A esa disciplina de ingenieria la denominaron **Harness Engineering**.

Yo comence a construir **AgentsMesh** hace poco mas de 50 dias. 52 dias, 600 commits, 965,687 lineas de codigo procesadas, 356,220 lineas de codigo actualmente en produccion. Una sola persona.

Pero lo mas interesante no son las cifras, sino la estructura del proyecto en si: utilice Harness Engineering para construir una herramienta de Harness Engineering.

El repositorio es completamente open source y el historial de git es publico. Todos los numeros se pueden verificar con git log.

## El entorno de ingenieria determina el techo de calidad del agente

52 dias de trabajo real me convencieron de que la calidad del output de un agente no depende solo del agente, sino fundamentalmente del terreno de ingenieria donde opera. Estas son decisiones que se sedimentaron en el codigo real.

### Arquitectura por capas: que el agente sepa donde modificar

El codebase sigue un **DDD** estricto: la capa domain contiene solo estructuras de datos, la capa service solo logica de negocio, la capa handler solo transformaciones de formato HTTP. 22 modulos de dominio con fronteras claras; cada modulo tiene su interface.go definiendo el contrato publico.

Cuando un agente necesita agregar una funcionalidad, sabe: estructuras de datos en domain, reglas de negocio en service, rutas en handler. En un codebase con fronteras difusas, el agente coloca las cosas en el lugar equivocado; en un codebase con fronteras claras, el codigo generado encaja de forma natural. Esto no es limpieza arquitectonica teorica: es el mapa de navegacion que el agente usa al generar codigo.

### Estructura de directorios como documentacion

La nomenclatura esta alineada de extremo a extremo. Tomemos Loop como ejemplo: backend/internal/domain/loop/ para estructuras de datos, backend/internal/service/loop/ para logica de negocio, web/src/components/loops/ para componentes frontend. El mapeo entre concepto de producto y ruta de codigo es directo: no hace falta buscar, el nombre del directorio es el mapa.

Los 16 modulos de dominio del backend (agentpod, channel, ticket, loop, runner...) se reflejan 1:1 en la capa service; los componentes del frontend se organizan por funcionalidad de producto (pod, tickets, loops, mesh, workspace), alineados con la nomenclatura del backend. Cuando un agente recibe una tarea relacionada con Ticket, no necesita explorar todo el codebase: basta con mirar la estructura de directorios para saber donde actuar.

Esta convencion no salio de un documento de estandares: se refuerza continuamente con cada commit generado por los agentes.

### La deuda tecnica se amplifica exponencialmente con agentes

Este fue uno de los hallazgos mas contraintuitivos de los 52 dias.

Si en algun modulo haces un atajo temporal — saltarte la capa service para consultar la base de datos directamente, o usar un valor hardcodeado — el agente aprende ese patron. La proxima vez que genere codigo similar, reutilizara ese "precedente". No es un evento aislado, es replicacion sistematica. La deuda tecnica deja de ser un problema local y empieza a propagarse.

Un ingeniero humano, al encontrar codigo cuestionable, suele pensar "esto es una trampa, mejor lo evito". El agente no hace ese juicio: ve un patron existente en el codebase y lo trata como practica valida.

Esto significa que la senal de calidad del codigo importa mucho mas que cuando solo escriben humanos. Si las buenas practicas son la norma, el agente amplifica buenas practicas; si los atajos temporales son la norma, el agente amplifica la deuda tecnica.

En la practica, me detuve varias veces a mitad de camino exclusivamente para limpiar deuda tecnica: cero funcionalidades nuevas, solo refactorizacion. No por estetica, sino para mantener la pureza de las senales de ingenieria en el repositorio. Este es un coste de mantenimiento especifico del desarrollo con agentes, y una de las diferencias mas grandes respecto al ritmo de desarrollo tradicional.

### Tipado fuerte como control de calidad en tiempo de compilacion

Go + TypeScript + Proto. El tipado fuerte desplaza una enorme cantidad de errores del runtime al tiempo de compilacion.

El agente genera una funcion con firma incompatible? Fallo de compilacion. Modifica un formato de API sin actualizar la definicion de tipos? TypeScript lo detecta al instante. Cambia el formato de mensajes del Runner sin sincronizar el Backend? El codigo generado por Proto no compila.

En lenguajes de tipado debil, estos errores se cuelan silenciosamente al runtime. El tipado fuerte los bloquea antes del commit. Cuanto mas corto el ciclo de feedback, mayor la eficiencia iterativa del agente.

### Cuatro capas de feedback en bucle cerrado

El agente necesita saber rapidamente que hizo mal. Una capa no basta; cuatro es el punto justo. Ademas, cuanto mas corto y preciso el bucle de feedback, mejor el resultado final del agente.

Primera capa: compilacion. Hot reload con Air: el codigo Go se reinicia en menos de 1 segundo tras un cambio; TypeScript marca errores de tipo en tiempo real. Los errores de sintaxis y tipado se eliminan aqui.

Segunda capa: tests unitarios. Mas de 700 tests cubren las capas domain y service. En 5 minutos el agente sabe si introdujo una regresion, especialmente en condiciones de borde como el aislamiento multi-tenant, que los agentes suelen pasar por alto.

Tercera capa: tests e2e. Validacion de flujos funcionales completos. Cubren fronteras de integracion que los tests unitarios no alcanzan: la interaccion real entre multiples modulos.

Cuarta capa: CI pipeline. Cada PR ejecuta automaticamente la suite completa de tests, linting, type-check y validacion de build multiplataforma. La ultima red de seguridad antes del merge, ejecutada por maquinas, sin depender de la atencion del revisor.

Las cuatro capas tienen latencia creciente y cubren categorias de errores cada vez mas amplias. Un cambio de una linea se valida en la primera capa; una refactorizacion transversal necesita la cuarta para una verificacion completa.

### Worktrees nativos para paralelismo

dev.sh calcula automaticamente offsets de puertos basados en el nombre del Git worktree, asignando un rango de puertos independiente a cada worktree. Multiples agentes trabajan simultaneamente en diferentes worktrees con entornos completamente aislados, sin necesidad de gestionar conflictos de puertos manualmente.

Esto es la extension del primitivo de aislamiento Pod al entorno de desarrollo: la misma logica, desde el entorno de ejecucion del agente hasta el entorno de desarrollo del agente.

### El codebase es el contexto del agente, no solo el prompt

Si se conectan todos estos puntos, convergen en una misma conclusion: el codebase en si es el contexto mas importante cuando el agente trabaja. La arquitectura por capas le dice donde modificar; la estructura de directorios le dice que archivo buscar; el nivel de limpieza de la deuda tecnica determina si aprende patrones buenos o malos; la densidad de tests determina cuanto se atreve a refactorizar; el tipado fuerte determina cuan pronto se detectan sus errores.

Esto significa que no necesitas construir un sistema de contexto externo al codebase: no hace falta Context Engineering explicito, ni un RAG dedicado, ni archivos de memoria adicionales. Lo que necesitas es que el codebase mismo sea un contexto de alta calidad. **El repositorio es el contexto.**

**Por eso la inversion en Harness Engineering coincide con la buena ingenieria de software de siempre**: escribir codigo claro, mantener buena arquitectura, limpiar la deuda tecnica a tiempo. La unica diferencia es el proposito: antes era para que los ingenieros humanos pudieran mantener el sistema; ahora tambien es para que los agentes de IA puedan trabajar de forma confiable.

## El ancho de banda cognitivo es una restriccion de ingenieria real

Alrededor del quinto dia, choque contra un muro real: 50,000 lineas de codigo procesadas por dia.

Tres worktrees abiertos simultaneamente, tres agentes ejecutando, yo alternando entre ellos para tomar decisiones. Al agregar un cuarto, la calidad de las decisiones caia visiblemente. No era una sensacion subjetiva: despues descubri que ese periodo dejo varias decisiones arquitectonicas deficientes.

Las 50,000 lineas diarias no son un limite de las herramientas, sino el techo natural del ancho de banda cognitivo humano. Puedes tomar decisiones arquitectonicas reales para aproximadamente tres flujos de trabajo en paralelo; mas alla de eso, la calidad se degrada.

La unica forma de romper ese techo: escalar mediante delegacion. No asignar mas tareas al agente, sino delegar la toma de decisiones misma. Que los agentes coordinen a otros agentes, y tu asciendes un nivel: de supervisar agentes individuales a supervisar el sistema que supervisa agentes. Asi nacio el modo **Autopilot**.

Esta es la intencion de diseno central de AgentsMesh. Y fue construyendolo con el mismo sistema que realmente lo comprendi.

## El colapso del coste de ensayo y error: hora de actualizar la metodologia

La arquitectura Relay de AgentsMesh no se diseno en una pizarra. La forjo el entorno de produccion.

Tres Pods ejecutandose simultaneamente tumbaron el Backend. Vi el crash, entendi la causa, reconstrui. Anade Relay para aislar el trafico de terminal. Surgen nuevos problemas: agregacion inteligente, gestion de conexiones bajo demanda. La arquitectura final emerge de fallos reales sucesivos, no de sesiones de diseno.

El viejo instinto de ingenieria dice: primero disena, luego construye. Analiza exhaustivamente los casos limite, porque equivocarse es caro.

Cuando el coste del ensayo y error se aproxima a cero, ese instinto se convierte en un lastre.

Aquel fallo de Relay tardo menos de dos dias en descubrirse y resolverse. En un equipo tradicional, habria sido dos semanas de discusion arquitectonica, y la discusion inevitablemente habria pasado algo por alto.

**Lo que la IA cambia no es la velocidad de escritura de codigo, sino la estructura de costes de todo el proceso de ingenieria.** Cuando iterar es suficientemente barato, el desarrollo guiado por experimentacion produce mejor arquitectura que el desarrollo guiado por diseno. Y mas rapido. El criterio de correccion arquitectonica deja de ser "aprobado en revision" y pasa a ser "sobrevivio en produccion".

## Validacion por autogeneracion

La tesis central de AgentsMesh: los agentes de IA pueden, bajo un Harness estructurado, colaborar para completar tareas de ingenieria complejas.

Yo use AgentsMesh para construir AgentsMesh.

Es la prueba mas directa de esa tesis. Si Harness Engineering realmente funciona, la herramienta deberia ser capaz de construirse a si misma.

52 dias, 965,687 lineas de codigo procesadas, 356,220 lineas de codigo en produccion, 600 commits, un solo autor.

OpenAI fue un equipo completo y les tomo cinco meses. No es una comparacion — escenarios diferentes, escalas diferentes. Pero hay algo que comparten: el Harness hace posible un output que antes era impensable.

El historial de commits es la evidencia. Cualquier ingeniero puede clonar el repositorio, ejecutar git log --numstat, y las cifras no cambian segun quien las mire.

## Tres primitivos de ingenieria

52 dias de practica y validacion por autogeneracion convergieron en tres primitivos de ingenieria. No son un framework de producto disenado de antemano; los forjaron problemas de ingenieria reales.

**Aislamiento** (Isolation)
Cada agente necesita su propio espacio de trabajo independiente. No es una buena practica: es un requisito duro. Sin aislamiento, el trabajo en paralelo es estructuralmente imposible. AgentsMesh lo implementa mediante **Pod**: cada agente opera en su propio Git worktree y sandbox. Los conflictos pasan de "podrian ocurrir" a "estructuralmente no pueden ocurrir". Y el aislamiento implica tambien cohesion: dentro del entorno aislado del Pod se prepara todo el contexto que el agente necesita para ejecutar — Repo, Skills, MCP y mas. En la practica, construir un Pod es el proceso de preparar el entorno de ejecucion del agente.

**Descomposicion** (Decomposition)
Los agentes no son buenos con "arregla este codebase". Son buenos con: eres dueno de este alcance, estos son los criterios de aceptacion, esta es la definicion de completado. La propiedad no es solo asignacion de tareas: cambia la forma en que el agente razona. La descomposicion es el trabajo de ingenieria que debe completarse antes de que cualquier agente se ejecute.

AgentsMesh ofrece dos abstracciones para la descomposicion: **Ticket** representa unidades de trabajo puntuales — desarrollo de funcionalidades, correccion de bugs, refactorizacion, con flujo completo de estados tipo kanban y vinculacion a MR; **Loop** representa tareas automatizadas periodicas — tests diarios, builds programados, escaneos de calidad de codigo, programados con expresiones Cron, donde cada ejecucion genera un registro LoopRun independiente. Las fronteras son claras: hacer algo una vez, usa Ticket; hacerlo repetidamente, usa Loop.

**Coordinacion** (Coordination)
No utilizamos abstracciones de roles para organizar la colaboracion entre agentes. Los equipos tradicionales necesitan roles porque cada persona domina solo unas pocas areas — un ingeniero frontend no escribe backend, un product manager no escribe codigo. Pero los agentes no tienen esa restriccion: el mismo agente puede escribir codigo, generar documentacion, analizar competidores, ejecutar tests, revisar PRs e incluso orquestar los workflows de otros agentes. Sus limites de capacidad no son fijos, se configuran mediante contexto y herramientas. Por eso la colaboracion entre agentes no necesita emular la division del trabajo humana; necesita comunicacion y permisos.

**Channel** resuelve la coordinacion a nivel colectivo: multiples Pods en un mismo espacio de colaboracion comparten mensajes, decisiones y documentos. Es la base para que un agente Supervisor y agentes Worker formen una estructura de colaboracion — no es un grupo de chat, es una capa de comunicacion estructurada con historial contextual.

**Binding** resuelve la coordinacion a nivel de capacidad: autorizacion punto a punto entre dos Pods. **pod:read** permite que un agente observe la salida de terminal de otro agente; **pod:write** permite que un agente controle directamente la ejecucion de otro. Binding es el mecanismo para que los agentes coordinen agentes — el Supervisor no percibe el estado del Worker enviando mensajes, sino observando directamente su terminal.

OpenAI llama a conceptos equivalentes Context Engineering, restricciones arquitectonicas y gestion de entropia. Los nombres son diferentes, pero el problema que resuelven es el mismo.

## Open source

Harness Engineering es una disciplina de ingenieria, no una funcionalidad de producto. En lugar de guardarlo, preferimos compartirlo.

Elegimos hacer open source AgentsMesh. Cuando construimos lo que podria ser una herramienta de ingenieria efectiva, el objetivo nunca fue "poseer el codigo", sino permitir que mas personas construyan herramientas de ingenieria aun mejores sobre esta base. En vez de encerrar practicas de ingenieria potencialmente correctas dentro de un producto, las abrimos para que la comunidad las valide, las evolucione y las supere.

Codigo en [GitHub](https://github.com/AgentsMesh/AgentsMesh)

Puedes usarlo para: desplegar tu propio Runner y ejecutar agentes de IA en entornos locales aislados; gestionar workflows de agentes con Ticket y Loop; coordinar multiples agentes en tareas complejas mediante Channel y Binding.

Si en tu propia practica de Harness Engineering has descubierto algo valioso, te invitamos a compartirlo en [GitHub Discussions](https://github.com/AgentsMesh/AgentsMesh/discussions) o directamente en un [Issue](https://github.com/AgentsMesh/AgentsMesh/issues). Este proyecto se construyo con agentes, y deberia seguir evolucionando impulsado por agentes e ingenieros por igual.
