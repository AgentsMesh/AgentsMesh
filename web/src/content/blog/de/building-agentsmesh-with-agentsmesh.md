---
title: 'AgentsMesh mit AgentsMesh gebaut: 52 Tage Harness Engineering als Ein-Personen-Projekt'
excerpt: "OpenAI nennt es Harness Engineering. Mit genau dieser Methodik hat eine einzelne Person in 52 Tagen, 600 Commits und 965.687 Zeilen Code-Durchsatz das Werkzeug gebaut, das Harness Engineering selbst ermöglicht. Die Codebasis ist der Kontext. Die Engineering-Umgebung bestimmt die Obergrenze der Agent-Qualität."
date: "2026-03-04"
author: "AgentsMesh Team"
category: "Insight"
readTime: 12
---

OpenAI hat kürzlich einen Artikel veröffentlicht, in dem beschrieben wird, wie sie mit KI-Agents in fünf Monaten über eine Million Zeilen Code produziert haben. Sie nennen diese Engineering-Praxis **Harness Engineering**.

Ich habe vor gut 50 Tagen begonnen, **AgentsMesh** zu bauen. 52 Tage, 600 Commits, 965.687 Zeilen Code-Durchsatz, 356.220 Zeilen Produktionscode. Eine Person.

Aber die eigentlich bemerkenswerte Aussage sind nicht die Zahlen, sondern die Struktur des Vorhabens selbst: Ich habe mit der Methodik des Harness Engineering ein Werkzeug für Harness Engineering gebaut.

Das Repository ist vollständig Open Source, die Git History öffentlich einsehbar. Sämtliche Zahlen lassen sich per git log verifizieren.

## Die Engineering-Umgebung bestimmt die Obergrenze der Agent-Qualität

52 Tage praktische Arbeit haben eine zentrale Erkenntnis bestätigt: Die Qualität der Agent-Ausgabe hängt nicht nur vom Agent selbst ab, sondern maßgeblich von der Engineering-Umgebung, in der er arbeitet. Die folgenden Punkte sind keine theoretischen Überlegungen, sondern reale, in der Codebasis verankerte Entscheidungen.

### Schichtenarchitektur: Dem Agent zeigen, wo Änderungen hingehören

Die Codebasis folgt einer strikten **DDD**-Schichtung: Die Domain-Schicht enthält ausschließlich Datenstrukturen, die Service-Schicht ausschließlich Geschäftslogik, die Handler-Schicht ausschließlich HTTP-Formatkonvertierung. 22 Domain-Module mit klaren Grenzen, jedes mit einer interface.go, die den Vertrag nach außen explizit definiert.

Wenn ein Agent eine neue Funktion hinzufügen muss, weiß er: Datenstrukturen gehören in domain, Geschäftsregeln in service, Routen in handler. In einer Codebasis mit unscharfen Grenzen platziert der Agent Code an der falschen Stelle; in einer Codebasis mit klaren Grenzen fügt sich der generierte Code natürlich ein. Das ist keine akademische Architekturübung -- es ist die Navigationskarte für die Code-Generierung.

### Verzeichnisstruktur als Dokumentation

Die Benennung ist über alle Schichten hinweg konsistent. Am Beispiel Loop: backend/internal/domain/loop/ für Datenstrukturen, backend/internal/service/loop/ für Geschäftslogik, web/src/components/loops/ für Frontend-Komponenten. Die Abbildung von Produktkonzept auf Codepfad ist direkt -- keine Suche nötig, der Verzeichnisname ist die Karte.

Die 16 Domain-Module im Backend (agentpod, channel, ticket, loop, runner, ...) spiegeln die Service-Schicht 1:1; die Web-Komponenten sind nach Produktfunktionen gegliedert (pod, tickets, loops, mesh, workspace) und an die Backend-Domain-Benennung angelehnt. Ein Agent, der eine Ticket-bezogene Aufgabe erhält, muss nicht die gesamte Codebasis durchsuchen -- die Verzeichnisstruktur allein zeigt, wo Änderungen nötig sind.

Diese Konvention ist nicht in einer Dokumentation festgehalten worden. Sie wurde durch jeden einzelnen Agent-Commit in der gesamten Codebasis kontinuierlich verstärkt.

### Technische Schulden werden durch Agents exponentiell verstärkt

Dies war eine der kontraintuitivsten Erkenntnisse der 52 Tage.

Wenn Sie in einem Modul einen temporären Kompromiss eingehen -- die Service-Schicht umgehen und direkt auf die Datenbank zugreifen oder eine hartcodierte Magic Number verwenden -- übernimmt der Agent dieses Muster. Beim nächsten Mal, wenn er ähnliche Funktionalität generiert, verwendet er diesen "Präzedenzfall" wieder. Nicht vereinzelt, sondern systematisch. Technische Schulden sind dann nicht mehr isoliert, sie beginnen sich auszubreiten.

Menschliche Ingenieure erkennen bei schlechtem Code in der Regel: "Das ist eine Altlast, die umgehe ich." Ein Agent trifft diese Unterscheidung nicht -- er sieht: In dieser Codebasis existiert dieses Muster, also ist es eine gültige Vorgehensweise.

Das bedeutet: Die Qualitätssignale im Repository sind bei Agent-gestützter Entwicklung weitaus wichtiger als beim menschlichen Programmieren. Gute Engineering-Praxis als Grundton -- der Agent verstärkt gute Engineering-Praxis. Temporäre Kompromisse als Grundton -- der Agent verstärkt technische Schulden.

Die praktische Konsequenz: Mehrfach wurde die Entwicklung unterbrochen, um gezielt technische Schulden zu beseitigen -- keine neuen Features, ausschließlich Refactoring. Nicht um den Code "hübscher" zu machen, sondern um die Reinheit der Engineering-Signale im Repository zu wahren. Das ist ein Agent-spezifischer Wartungsaufwand und einer der größten Unterschiede zum traditionellen Entwicklungsrhythmus.

### Starke Typisierung als Qualitätsschranke zur Compile-Zeit

Go + TypeScript + Proto. Starke Typisierung verlagert eine Vielzahl von Fehlern von der Laufzeit in die Compile-Zeit.

Der Agent generiert eine Funktion mit falscher Signatur? Kompilierungsfehler. Der Agent ändert ein API-Format, vergisst aber die Typdefinition zu aktualisieren? TypeScript meldet den Fehler sofort. Der Agent ändert das Nachrichtenformat des Runners, synchronisiert aber nicht das Backend? Der aus Proto generierte Code kompiliert nicht.

In schwach typisierten Sprachen würden diese Fehler unbemerkt in die Laufzeit gelangen. Starke Typisierung fängt sie vor dem Commit ab. Je kürzer die Feedback-Schleife, desto höher die Iterationseffizienz des Agents.

### Vierstufige Feedback-Schleife

Ein Agent muss schnell erfahren, was er falsch gemacht hat. Eine Stufe reicht nicht, vier sind genau richtig. Und je kürzer und präziser die Feedback-Schleife, desto besser das Ergebnis.

Stufe 1: Kompilierung. Air Hot-Reload, Go-Code startet innerhalb einer Sekunde nach Änderung neu; TypeScript-Typfehler werden in Echtzeit angezeigt. Syntax- und Typfehler werden auf dieser Stufe eliminiert.

Stufe 2: Unit-Tests. Über 700 Tests decken Domain- und Service-Schicht ab. Der Agent erfährt innerhalb von fünf Minuten nach einer Änderung, ob eine Regression eingeführt wurde -- insbesondere bei Randbedingungen wie Mandantenisolierung, die Agents gerne übersehen.

Stufe 3: End-to-End-Tests. Verifizierung realer Funktionspfade. Abdeckung von Integrationsgrenzen, die in Unit-Tests nicht erfasst werden -- das tatsächliche Zusammenspiel mehrerer Module.

Stufe 4: CI Pipeline. Jeder PR durchläuft automatisch die vollständige Testsuite, Linting, Type-Checking und Multiplattform-Build-Verifikation. Das letzte Sicherheitsnetz vor dem Merge -- maschinell ausgeführt, unabhängig von der Sorgfalt des Reviewers.

Die vier Stufen haben zunehmende Latenz bei zunehmender Fehlerabdeckung. Eine einzelne Codezeile wird auf Stufe 1 validiert; ein modulübergreifendes Refactoring kann erst auf Stufe 4 vollständig verifiziert werden.

### Worktree-native Parallelisierung

dev.sh berechnet automatisch Port-Offsets basierend auf dem Git-Worktree-Namen und weist jedem Worktree einen eigenen Portbereich zu. Mehrere Agents arbeiten gleichzeitig in verschiedenen Worktrees bei vollständiger Umgebungsisolierung -- keine manuelle Portkonfliktverwaltung erforderlich.

Das ist die konsequente Fortsetzung des Pod-Isolierungsprinzips auf der Ebene der Entwicklungsumgebung -- dieselbe Logik, von der Agent-Ausführungsumgebung bis zur Agent-Entwicklungsumgebung.

### Die Codebasis ist der Kontext des Agents -- nicht nur der Prompt

Betrachtet man alle genannten Punkte zusammen, zeigt sich ein gemeinsames Ergebnis: Die Codebasis selbst ist der wichtigste Kontext, in dem der Agent arbeitet. Die Schichtenarchitektur sagt dem Agent, wo er ändern soll; die Verzeichnisstruktur sagt ihm, welche Dateien relevant sind; der Bereinigungsgrad technischer Schulden bestimmt, ob der Agent gute oder schlechte Muster lernt; die Testdichte bestimmt, wie mutig der Agent refactoren kann; die starke Typisierung bestimmt, wie früh Fehler erkannt werden.

Das bedeutet: Sie müssen kein separates Kontextsystem außerhalb der Codebasis aufbauen -- kein aufwendiges Context Engineering, kein separates RAG-System, keine zusätzlichen Memory-Dateien. Was Sie tun müssen, ist die Codebasis selbst zu einem hochwertigen Kontext zu machen. **Das Repository ist der Context.**

**Deshalb stimmt die Investitionsrichtung von Harness Engineering mit klassischem Software Engineering überein**: sauberen Code schreiben, gute Architektur pflegen, technische Schulden zeitnah beseitigen. Der einzige Unterschied liegt im Zweck -- früher ging es darum, die Wartbarkeit für menschliche Ingenieure zu erhöhen; heute dient es gleichzeitig dazu, KI-Agents verlässlich arbeiten zu lassen.

## Kognitive Bandbreite als reale Engineering-Constraint

Um den fünften Tag herum traf ich auf eine reale Grenze: rund 50.000 Zeilen täglicher Code-Durchsatz.

Drei Worktrees gleichzeitig geöffnet, drei Agents in Betrieb, ich wechselte zwischen ihnen, um Entscheidungen zu treffen. Bei einem vierten sank die Entscheidungsqualität merklich. Kein subjektiver Eindruck -- es zeigte sich erst später, dass in dieser Phase mehrere schlechte Architekturentscheidungen getroffen wurden.

50.000 Zeilen täglicher Durchsatz sind keine Werkzeugbeschränkung, sondern die natürliche Obergrenze menschlicher kognitiver Bandbreite. Sie können für etwa drei parallele Arbeitsströme tatsächlich tragfähige Architekturentscheidungen treffen. Darüber hinaus sinkt die Qualität.

Der einzige Weg, diese Grenze zu durchbrechen: Delegation statt Skalierung. Nicht dem Agent mehr Aufgaben geben, sondern die Entscheidungsfindung selbst delegieren. Agents koordinieren Agents, während Sie selbst eine Ebene aufsteigen -- von der Überwachung einzelner Agents zur Überwachung des Systems, das Agents überwacht. Daraus entstand der **Autopilot**-Modus.

Das ist die zentrale Designabsicht von AgentsMesh. Und es ist eine Erkenntnis, die ich erst im Prozess des Selbstbaus wirklich verstanden habe.

## Kollaps der Trial-and-Error-Kosten: Engineering-Methodik muss sich anpassen

Die Relay-Architektur von AgentsMesh wurde nicht am Whiteboard entworfen. Sie wurde in der Produktionsumgebung durch reale Ausfälle geformt.

Drei gleichzeitig laufende Pods brachten das Backend zum Absturz. Ich beobachtete den Ausfall, verstand die Ursache und baute neu. Relay wurde hinzugefügt, um Terminal-Traffic zu isolieren. Neue Probleme traten auf, intelligente Aggregation und bedarfsgesteuerte Verbindungsverwaltung kamen hinzu. Die finale Architektur entstand aus einer Folge realer Ausfälle -- nicht aus einer Whiteboard-Diskussion.

Die tradierte Engineering-Intuition lautet: erst entwerfen, dann bauen -- Randfälle gründlich durchspielen, weil Fehler teuer sind.

Wenn die Trial-and-Error-Kosten gegen null gehen, wird diese Intuition zur Bremse.

Der Relay-Ausfall wurde innerhalb von zwei Tagen von der Entdeckung bis zur Behebung gelöst. In einem traditionellen Team wäre das eine zweiwöchige Architekturdiskussion gewesen -- die mit Sicherheit etwas übersehen hätte.

**KI verändert nicht die Geschwindigkeit des Code-Schreibens, sondern die gesamte Kostenstruktur des Engineering-Prozesses.** Wenn Iteration hinreichend günstig ist, liefert experimentgetriebene Entwicklung bessere Architekturen als designgetriebene -- und das schneller. Der Maßstab für architektonische Korrektheit ist nicht das Bestehen eines Reviews, sondern das Bestehen der Produktionsumgebung.

## Selbst-Bootstrap als Validierung

Die zentrale These von AgentsMesh: KI-Agents können unter einem strukturierten Harness gemeinsam komplexe Engineering-Aufgaben bewältigen.

Ich habe AgentsMesh mit AgentsMesh gebaut.

Das ist die direkteste Prüfung dieser These. Wenn Harness Engineering tatsächlich funktioniert, muss dieses Werkzeug in der Lage sein, sich selbst zu bauen.

52 Tage, 965.687 Zeilen Code-Durchsatz, 356.220 Zeilen Produktionscode, 600 Commits, ein Autor.

OpenAI ist ein ganzes Team und hat fünf Monate gebraucht. Das ist kein direkter Vergleich -- die Szenarien und Maßstäbe sind verschieden. Aber eines haben beide gemeinsam: Der Harness macht Ergebnisse möglich, die ohne ihn undenkbar wären.

Die Commit History ist der Beweis. Jeder Ingenieur kann das Repository klonen und git log --numstat ausführen. Die Zahlen ändern sich nicht, unabhängig davon, wer sie prüft.

## Drei Engineering-Primitive

52 Tage Praxis und Selbst-Bootstrap-Validierung konvergierten schließlich zu drei Engineering-Primitiven. Das ist kein vorab entworfenes Produkt-Framework -- es wurde aus realen Engineering-Problemen destilliert.

**Isolation** (Isolation)
Jeder Agent benötigt seinen eigenen, unabhängigen Arbeitsbereich. Keine Best Practice, sondern eine harte Voraussetzung. Ohne Isolation ist paralleles Arbeiten strukturell unmöglich. AgentsMesh realisiert dies über **Pods**: Jeder Agent läuft in einem eigenen Git Worktree und einer eigenen Sandbox. Konflikte gehen von "können auftreten" zu "können strukturell nicht auftreten" über. Isolation bedeutet zugleich Kohäsion -- in der isolierten Pod-Umgebung wird der vollständige Kontext bereitgestellt, den der Agent für die Ausführung benötigt: Repository, Skills, MCP und mehr. Der Pod-Aufbau ist im Wesentlichen die Vorbereitung der Ausführungsumgebung für den Agent.

**Dekomposition** (Decomposition)
Agents sind nicht gut darin, "kümmere dich um diese Codebasis" zu verarbeiten. Was sie beherrschen: "Du bist für diesen Scope zuständig, hier sind die Abnahmekriterien, hier ist die Definition of Done." Ownership ist nicht nur Aufgabenzuweisung -- es verändert die Art, wie der Agent denkt. Dekomposition ist die Engineering-Arbeit, die vor jedem Agent-Lauf abgeschlossen sein muss.

AgentsMesh bietet zwei Abstraktionen für Dekomposition: **Tickets** für einmalige Arbeitspakete -- Feature-Entwicklung, Bugfixes, Refactoring, mit vollständigem Kanban-Statusfluss und MR-Verknüpfung; **Loops** für periodische, automatisierte Aufgaben -- tägliche Tests, geplante Builds, Code-Qualitätsscans, gesteuert über Cron-Ausdrücke, wobei jede Ausführung einen eigenen LoopRun-Datensatz hinterlässt. Die Grenze zwischen beiden Aufgabenformen ist klar: Eine Sache einmal erledigen -- Ticket. Dieselbe Sache wiederholt erledigen -- Loop.

**Koordination** (Coordination)
Wir verwenden keine Rollen-Abstraktion für die Agent-Koordination. Traditionelle Teams benötigen Stellenprofile, weil jede Person nur wenige Fachgebiete beherrscht -- Frontend-Entwickler schreiben kein Backend, Produktmanager schreiben keinen Code. Agents unterliegen dieser Beschränkung nicht: Derselbe Agent kann Code schreiben, Dokumentation generieren, Wettbewerbsanalysen durchführen, Tests ausführen, Pull Requests reviewen und sogar Workflows anderer Agents orchestrieren. Seine Fähigkeitsgrenzen sind nicht fest, sondern werden über Kontext und Tool-Konfiguration definiert. Agent-Koordination muss daher menschliche Arbeitsteilung nicht nachahmen -- sie benötigt Kommunikation und Berechtigungen.

**Channels** adressieren die kollektive Ebene: Mehrere Pods teilen sich einen Kollaborationsraum für Nachrichten, Entscheidungen und Dokumente. Das ist die Grundlage, auf der Supervisor-Agents und Worker-Agents Kooperationsstrukturen bilden können -- kein Gruppenchat, sondern eine strukturierte Kommunikationsschicht mit Kontexthistorie.

**Bindings** adressieren die Fähigkeitsebene: Punkt-zu-Punkt-Berechtigungen zwischen zwei Pods. **pod:read** ermöglicht einem Agent, die Terminalausgabe eines anderen Agents zu beobachten; **pod:write** ermöglicht einem Agent, die Ausführung eines anderen Agents direkt zu steuern. Bindings sind der Mechanismus, über den Agents andere Agents koordinieren -- der Supervisor erfasst den Zustand des Workers nicht durch Nachrichtenaustausch, sondern durch direkten Einblick in sein Terminal.

OpenAI nennt die entsprechenden Konzepte Context Engineering, Architektur-Constraints und Entropie-Management. Andere Begriffe, dasselbe Problem.

## Open Source

Harness Engineering ist eine Engineering-Disziplin, kein Produktfeature. Statt das Wissen für sich zu behalten, setzen wir auf Offenheit als Impulsgeber.

Wir haben uns entschieden, AgentsMesh als Open Source zu veröffentlichen. Wenn wir möglicherweise ein effektives Engineering-Werkzeug gebaut haben, war das Ziel nie, "den Code zu besitzen", sondern mehr Menschen zu ermöglichen, auf dieser Grundlage noch bessere Engineering-Werkzeuge zu entwickeln. Statt potenziell richtige Engineering-Praktiken in einem Produkt einzuschließen, machen wir sie offen -- damit die Community sie validieren, weiterentwickeln und übertreffen kann.

Der Code ist auf [GitHub](https://github.com/AgentsMesh/AgentsMesh)

Sie können damit: einen eigenen Runner bereitstellen und KI-Agents in lokalen, isolierten Umgebungen ausführen; mit Tickets und Loops die Workflows Ihrer Agents verwalten; über Channels und Bindings mehrere Agents zur Bewältigung komplexer Aufgaben koordinieren.

Wenn Sie bei Ihrer eigenen Harness-Engineering-Praxis Erkenntnisse gewonnen haben -- besuchen Sie die [GitHub Discussions](https://github.com/AgentsMesh/AgentsMesh/discussions) oder eröffnen Sie direkt ein [Issue](https://github.com/AgentsMesh/AgentsMesh/issues). Dieses Projekt wurde selbst mit Agents gebaut. Es sollte weiterhin gemeinsam von Agents und Ingenieuren weiterentwickelt werden.
