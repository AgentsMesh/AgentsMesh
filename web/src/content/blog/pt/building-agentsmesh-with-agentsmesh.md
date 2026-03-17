---
title: 'Construindo o AgentsMesh com o AgentsMesh: a prática de Harness Engineering de uma pessoa em 52 dias'
excerpt: "A OpenAI chamou de Harness Engineering. Com essa metodologia, uma pessoa em 52 dias, 600 commits e 965.687 linhas de throughput, construiu a própria ferramenta de Harness Engineering. O codebase é o contexto, e o ambiente de engenharia determina o teto de qualidade do Agent."
date: "2026-03-04"
author: "AgentsMesh Team"
category: "Insight"
readTime: 12
---

A OpenAI publicou recentemente um artigo descrevendo como usaram AI Agents para produzir mais de um milhão de linhas de código em 5 meses. Eles chamaram essa prática de engenharia de **Harness Engineering**.

Comecei a construir o **AgentsMesh** há pouco mais de 50 dias. 52 dias, 600 commits, 965.687 linhas de throughput de código, 356.220 linhas de código em produção. Uma pessoa.

Mas o mais interessante não são os números. É a estrutura da coisa em si: usei a abordagem de Harness Engineering para construir uma ferramenta de Harness Engineering.

O repositório é totalmente open source e o histórico do Git é público. Todos os números podem ser verificados com git log.

## O ambiente de engenharia determina o teto de qualidade do Agent

52 dias de trabalho real me convenceram de que a qualidade da produção de um Agent não depende apenas do Agent em si, mas principalmente do terreno de engenharia onde ele opera. Estas são escolhas que se sedimentaram de verdade no codebase.

### Arquitetura em camadas: o Agent sabe onde modificar

O codebase segue uma arquitetura **DDD** rigorosa: a camada domain contém apenas estruturas de dados, a camada service apenas lógica de negócios, a camada handler apenas conversão de formato HTTP. São 22 módulos de domínio com fronteiras claras, e cada módulo tem um interface.go que define explicitamente o contrato externo.

Quando o Agent precisa adicionar uma nova funcionalidade, ele sabe: estruturas de dados vão no domain, regras de negócio vão no service, rotas vão no handler. Em um codebase com fronteiras difusas, o Agent coloca as coisas no lugar errado; em um codebase com fronteiras claras, o código do Agent se integra naturalmente. Isso não é limpeza arquitetural teórica, é o mapa de navegação que o Agent usa ao gerar código.

### Estrutura de diretórios como documentação

A nomenclatura é totalmente alinhada entre frontend e backend. Tomando Loop como exemplo: backend/internal/domain/loop/ contém as estruturas de dados, backend/internal/service/loop/ contém a lógica de negócios, web/src/components/loops/ contém os componentes do frontend. O mapeamento do conceito de produto para o caminho no código é direto: sem necessidade de buscar, o nome do diretório já é o mapa.

Os 16 módulos de domain do backend (agentpod, channel, ticket, loop, runner...) espelham 1:1 a camada service; os components do frontend são organizados por funcionalidade de produto (pod, tickets, loops, mesh, workspace), alinhados com a nomenclatura do domain no backend. Um Agent que recebe uma tarefa relacionada a Ticket não precisa explorar todo o codebase; basta olhar os diretórios para saber onde atuar.

Essa convenção não foi escrita em documentação. Ela se reforça continuamente a cada commit do Agent ao longo de todo o codebase.

### Dívida técnica é amplificada exponencialmente pelo Agent

Esta foi uma das descobertas mais contraintuitivas dos 52 dias.

Quando você faz uma concessão temporária em algum módulo, como acessar o banco diretamente contornando a camada service, ou usar um número mágico hardcoded, o Agent aprende esse padrão. Na próxima vez que gera código para funcionalidade semelhante, ele reutiliza esse "precedente". Não é algo pontual; é replicação sistemática. A dívida técnica deixa de ser isolada e começa a se espalhar.

Engenheiros humanos, ao encontrar código ruim, geralmente sabem: "isso é uma armadilha, melhor desviar". O Agent não faz esse julgamento. Ele vê: esse padrão existe no codebase, logo é uma abordagem válida.

Isso significa que o sinal de qualidade no repositório é muito mais importante do que quando humanos escrevem código sozinhos. Se boas práticas de engenharia são a regra, o Agent amplifica boas práticas; se concessões temporárias são a regra, o Agent amplifica dívida técnica.

Na prática: parei várias vezes no meio do caminho especificamente para limpar dívida técnica. Sem lançar funcionalidades novas, apenas refatoração. Não para deixar o código "bonito", mas para manter a pureza do sinal de engenharia no repositório. Esse é um custo de manutenção específico do desenvolvimento colaborativo com Agents, e uma das maiores diferenças em relação ao ritmo de desenvolvimento tradicional.

### Tipagem forte como gate de qualidade em tempo de compilação

Go + TypeScript + Proto. Tipagem forte desloca uma enorme quantidade de erros do runtime para o tempo de compilação.

O Agent gerou uma função com assinatura incompatível? Falha de compilação. O Agent modificou o formato da API mas esqueceu de atualizar a definição de tipos? TypeScript reporta o erro diretamente. O Agent alterou o formato de mensagem do Runner sem sincronizar com o Backend? O código gerado pelo Proto não compila.

Esses erros, em linguagens de tipagem fraca, entrariam silenciosamente no runtime. Tipagem forte os barra antes do commit. Quanto mais curto o loop de feedback, maior a eficiência de iteração do Agent.

### Quatro camadas de feedback em loop fechado

O Agent precisa saber rapidamente o que fez de errado. Uma camada não basta; quatro é o ponto ideal. E quanto mais curto e preciso o loop de feedback, melhor o resultado entregue pelo Agent.

Primeira camada: compilação. Hot reload com Air, reinício do código Go em menos de 1 segundo após modificação; erros de tipo TypeScript marcados em tempo real. Erros de sintaxe e de tipo são eliminados nesta camada.

Segunda camada: testes unitários. Mais de 700 testes cobrindo as camadas domain e service. O Agent sabe em menos de 5 minutos se introduziu uma regressão, especialmente em condições de contorno como isolamento multi-tenant, que Agents facilmente ignoram.

Terceira camada: e2e. Validação ponta a ponta de fluxos funcionais reais. Cobre fronteiras de integração que os testes unitários não alcançam, com a interação real de múltiplos módulos.

Quarta camada: CI pipeline. Cada PR executa automaticamente o conjunto completo de testes, lint, type-check e verificação de build multiplataforma. A última rede de segurança antes do merge, executada por máquina, sem depender da atenção do revisor.

As quatro camadas têm latência crescente e cobertura de tipos de erro cada vez mais ampla. Quando o Agent altera uma linha de código, a primeira camada confirma; quando o Agent faz refatoração entre módulos, apenas a quarta camada consegue validar completamente.

### Worktree com suporte nativo a paralelismo

O dev.sh calcula automaticamente o offset de portas com base no nome do Git worktree, atribuindo intervalos de portas independentes para cada worktree. Múltiplos Agents trabalhando simultaneamente em diferentes worktrees têm ambientes completamente isolados, sem necessidade de gerenciar conflitos de porta manualmente.

Essa é a extensão da primitiva de isolamento do Pod na camada de ambiente de desenvolvimento: a mesma lógica, desde o ambiente de execução do Agent até o ambiente de desenvolvimento do Agent.

### O codebase é o contexto do Agent, não apenas o prompt

Reunindo tudo isso, percebe-se que apontam para a mesma conclusão: o codebase em si é o contexto mais importante durante o trabalho do Agent. A arquitetura em camadas diz ao Agent onde modificar; a estrutura de diretórios diz ao Agent qual arquivo procurar; o nível de limpeza da dívida técnica determina se o Agent aprende padrões bons ou ruins; a densidade de testes determina quanta coragem o Agent tem para refatorar; a tipagem forte determina quão cedo os erros do Agent são detectados.

Isso significa que você não precisa construir um sistema de contexto externo ao codebase. Não precisa fazer Context Engineering deliberado, não precisa montar um RAG separado, não precisa manter arquivos de memória extras. O que você precisa fazer é tornar o codebase em si um contexto de alta qualidade. **O repositório é o contexto.**

**Por isso o investimento em Harness Engineering e em engenharia de software tradicional convergem**: escrever código claro, manter boa arquitetura, limpar dívida técnica regularmente. A diferença é apenas o propósito: antes era para facilitar a manutenção por engenheiros humanos; agora é também para que AI Agents possam trabalhar de forma confiável.

## Largura de banda cognitiva é uma restrição real de engenharia

Por volta do quinto dia, eu bati em uma parede real: throughput diário de 50 mil linhas de código.

Três worktrees abertos simultaneamente, três Agents rodando, eu alternando entre eles para tomar decisões. Ao adicionar o quarto, a qualidade das decisões caiu visivelmente. Não foi sensação; foi comprovado depois, ao descobrir que aquele período deixou algumas decisões arquiteturais ruins.

O throughput diário de 50 mil linhas não é limitação da ferramenta: é o teto natural da largura de banda cognitiva humana. Você consegue tomar decisões arquiteturais reais para cerca de três fluxos de trabalho paralelos; acima disso, a qualidade começa a cair.

A única forma de romper essa barreira: trocar delegação por escala. Não dar mais tarefas ao Agent, mas delegar a própria tomada de decisão. Fazer Agents coordenarem Agents, e você subir um nível: de supervisionar um Agent individual para supervisionar o sistema que supervisiona Agents. Por isso criamos o modo **Autopilot**.

Essa é a intenção central de design do AgentsMesh. E algo que só entendi de verdade ao construí-lo com ele mesmo.

## Colapso do custo de tentativa e erro: a metodologia de engenharia precisa ser atualizada

A arquitetura Relay do AgentsMesh não foi projetada. Foi forjada pelo ambiente de produção.

Três Pods rodando simultaneamente derrubaram o Backend. Eu vi cair, entendi a causa, reconstruí. Adicionei o Relay para isolar o tráfego de terminal. Novos problemas surgiram; adicionei agregação inteligente, gerenciamento de conexão sob demanda. A arquitetura final veio de falhas reais sucessivas, não de discussões em quadro branco.

A intuição antiga de engenharia manda projetar antes de construir: analisar exaustivamente os casos extremos, porque o custo de errar é alto.

Quando o custo de tentativa e erro se aproxima de zero, essa intuição se torna um fardo.

Aquela falha do Relay levou menos de dois dias da descoberta à correção. Em uma equipe tradicional, seriam duas semanas de discussão arquitetural, e a discussão inevitavelmente deixaria algo escapar.

**O que a IA muda não é a velocidade de escrever código, é a estrutura de custos de todo o processo de engenharia.** Quando a iteração é suficientemente barata, experimentação supera design na produção de arquiteturas melhores, e mais rápido. O critério de correção arquitetural deixa de ser aprovação em review para ser sobrevivência em produção.

## Validação por auto-bootstrap

A proposta central do AgentsMesh: AI Agents podem, sob um Harness estruturado, colaborar para completar tarefas complexas de engenharia.

Eu usei o AgentsMesh para construir o AgentsMesh.

Essa é a verificação mais direta da proposta. Se Harness Engineering realmente funciona, a ferramenta deveria ser capaz de construir a si mesma.

52 dias, 965.687 linhas de throughput de código, 356.220 linhas de código em produção, 600 commits, um autor.

A OpenAI usou uma equipe inteira e levou 5 meses. Não é uma comparação direta: cenários diferentes, escalas diferentes. Mas uma coisa é igual: o Harness torna possível uma produção que antes seria impossível.

O histórico de commits é a evidência. Qualquer engenheiro pode clonar o repositório, rodar git log --numstat, e os números não mudam dependendo de quem olha.

## Três primitivas de engenharia

52 dias de prática e validação por auto-bootstrap convergiram em três primitivas de engenharia. Não foram um framework de produto pré-desenhado; foram forçadas por problemas reais de engenharia.

**Isolamento** (Isolation)
Cada Agent precisa de seu próprio espaço de trabalho independente. Não é best practice: é pré-requisito inegociável. Sem isolamento, trabalho paralelo é estruturalmente impossível. O AgentsMesh implementa isso com **Pods**: cada Agent roda em um Git worktree e sandbox independentes. Conflitos passam de "podem acontecer" para "estruturalmente impossíveis". E isolamento também significa coesão: no ambiente independente do Pod, todo o contexto necessário para a execução do Agent é preparado: Repo, Skills, MCP e mais. Na prática, o processo de construir o Pod é o processo de preparar o ambiente para a execução do Agent.

**Decomposição** (Decomposition)
Agents não lidam bem com "me ajuda com esse codebase". O que funciona é: você é dono deste escopo, estes são os critérios de aceitação, esta é a definição de pronto. Ownership não é apenas atribuição de tarefas; muda a forma como o Agent raciocina. Decomposição é o trabalho de engenharia que precisa estar pronto antes de qualquer Agent rodar.

O AgentsMesh oferece duas abstrações para decomposição: **Ticket** corresponde a itens de trabalho pontuais (desenvolvimento de feature, correção de bug, refatoração) com fluxo completo de status em kanban e associação com MR; **Loop** corresponde a tarefas automatizadas recorrentes (testes diários, builds agendados, varredura de qualidade de código) com agendamento via expressão Cron, cada execução gerando um registro de LoopRun independente. As fronteiras são claras: para fazer algo uma vez, use Ticket; para fazer a mesma coisa repetidamente, use Loop.

**Coordenação** (Coordination)
Não usamos abstração de cargos para organizar a colaboração entre Agents. Equipes tradicionais precisam de funções porque cada pessoa domina poucas especialidades: engenheiro de frontend não escreve backend, product manager não escreve código. Mas Agents não têm essa restrição: o mesmo Agent pode escrever código, gerar documentação, fazer análise competitiva, executar testes, revisar PRs e até orquestrar workflows de outros Agents. Seus limites de capacidade não são fixos; são configurados via contexto e ferramentas. Portanto, a colaboração entre Agents não precisa simular a divisão de trabalho humana. Precisa de comunicação e permissões.

**Channel** resolve a camada coletiva: múltiplos Pods compartilham mensagens, decisões e documentos no mesmo espaço colaborativo. É a base para que Supervisor Agents e Worker Agents formem estruturas de colaboração: não é um grupo de chat, é uma camada de comunicação estruturada com histórico contextual.

**Binding** resolve a camada de capacidades: autorização ponto a ponto entre dois Pods. **pod:read** permite que um Agent observe a saída de terminal de outro Agent; **pod:write** permite que um Agent controle diretamente a execução de outro. Binding é o mecanismo de Agent coordenando Agent: o Supervisor não depende de mensagens para perceber o estado do Worker, ele vê diretamente o terminal.

A OpenAI chama os equivalentes de Context Engineering, restrições arquiteturais e gerenciamento de entropia. Nomes diferentes, mesmo problema.

## Open source

Harness Engineering é uma disciplina de engenharia, não uma funcionalidade de produto. Em vez de guardar para nós mesmos, preferimos colocar a primeira pedra para que outros construam algo maior.

Escolhemos tornar o AgentsMesh open source. Quando o que estamos construindo pode ser uma ferramenta de engenharia eficaz, o objetivo nunca foi "possuir o código", mas permitir que mais pessoas construam ferramentas de engenharia ainda melhores a partir daqui. Em vez de trancar práticas possivelmente corretas dentro de um produto, melhor abri-las para que a comunidade valide, evolua e supere.

O código está no [GitHub](https://github.com/AgentsMesh/AgentsMesh)

Você pode usá-lo para: implantar seu próprio Runner e rodar AI Agents em ambientes locais isolados; gerenciar workflows de Agents com Tickets e Loops; fazer múltiplos Agents colaborarem em tarefas complexas via Channels e Bindings.

Se você fez descobertas na sua própria prática de Harness Engineering, venha trocar ideia no [GitHub Discussions](https://github.com/AgentsMesh/AgentsMesh/discussions) ou abra uma [Issue](https://github.com/AgentsMesh/AgentsMesh/issues). Este projeto foi construído com Agents, e deve continuar evoluindo com Agents e engenheiros juntos.
