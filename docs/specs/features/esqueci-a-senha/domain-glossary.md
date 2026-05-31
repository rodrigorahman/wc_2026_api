# Glossário de Domínio — Esqueci a senha

## Termos

**Senha temporária**:
Credencial adicional, gerada pelo sistema por fonte criptográfica e de tempo limitado (15 min), que coexiste com a senha vigente durante a janela de recuperação — não a substitui.
_Evitar_: token de reset, código de uso único, OTP, senha provisória

**Troca obrigatória**:
Sinalização (`password_change_required`) retornada no Login quando o acesso foi concedido pela senha temporária, indicando que o usuário deve definir uma nova senha definitiva. É sinalização, não enforcement de backend.
_Evitar_: reset forçado, senha pendente

**Recuperação de acesso**:
Fluxo completo iniciado pelo pedido (`RequestPasswordRecovery`) que emite a senha temporária por e-mail e termina na troca definitiva (`ChangePassword`).
_Evitar_: reset de senha, redefinição

## Relacionamentos
- Uma **Recuperação de acesso** emite exatamente uma **Senha temporária** ativa por **User** (um novo pedido sobrescreve a anterior).
- Logar com a **Senha temporária** dispara a **Troca obrigatória**; logar com a senha vigente não.
- A **Troca obrigatória** (via `ChangePassword`) substitui a senha vigente e invalida a **Senha temporária**.

## Ambiguidades resolvidas
- "senha temporária" **não** é um token de reset nem OTP — a recuperação por token/OTP foi explicitamente podada (PRD §4.2 / tech-alignment D0). É uma credencial de login válida que coexiste com a senha original (tech-alignment D4).
- "troca obrigatória" é **sinalização** ao cliente, não um gate de backend que bloqueia operações (decisão de challenge 2026-05-31).
