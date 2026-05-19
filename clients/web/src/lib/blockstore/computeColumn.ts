import type { JSONMap } from "@/lib/api/blockstoreTypes";

export function computeColumn(expr: string, data: JSONMap): number | null {
  try {
    const tokens = tokenize(expr);
    const parser = new Parser(tokens, data);
    const value = parser.parseExpression();
    if (!parser.atEnd()) return null;
    return Number.isFinite(value) ? value : null;
  } catch {
    return null;
  }
}

type Token =
  | { kind: "num"; value: number }
  | { kind: "ref"; key: string }
  | { kind: "op"; op: "+" | "-" | "*" | "/" | "(" | ")" };

function tokenize(expr: string): Token[] {
  const out: Token[] = [];
  let i = 0;
  while (i < expr.length) {
    const c = expr[i];
    if (c === " " || c === "\t" || c === "\n") {
      i += 1;
      continue;
    }
    if (c === "{") {
      const end = expr.indexOf("}", i);
      if (end < 0) throw new Error("unterminated ref");
      out.push({ kind: "ref", key: expr.slice(i + 1, end) });
      i = end + 1;
      continue;
    }
    if (/[0-9.]/.test(c)) {
      let j = i + 1;
      while (j < expr.length && /[0-9.]/.test(expr[j])) j += 1;
      out.push({ kind: "num", value: Number(expr.slice(i, j)) });
      i = j;
      continue;
    }
    if (c === "+" || c === "-" || c === "*" || c === "/" || c === "(" || c === ")") {
      out.push({ kind: "op", op: c });
      i += 1;
      continue;
    }
    throw new Error(`unexpected char: ${c}`);
  }
  return out;
}

class Parser {
  private pos = 0;
  constructor(private tokens: Token[], private data: JSONMap) {}

  atEnd(): boolean {
    return this.pos >= this.tokens.length;
  }

  private peek(): Token | undefined {
    return this.tokens[this.pos];
  }

  parseExpression(): number {
    let value = this.parseTerm();
    while (!this.atEnd()) {
      const t = this.peek()!;
      if (t.kind !== "op" || (t.op !== "+" && t.op !== "-")) break;
      this.pos += 1;
      const rhs = this.parseTerm();
      value = t.op === "+" ? value + rhs : value - rhs;
    }
    return value;
  }

  private parseTerm(): number {
    let value = this.parseFactor();
    while (!this.atEnd()) {
      const t = this.peek()!;
      if (t.kind !== "op" || (t.op !== "*" && t.op !== "/")) break;
      this.pos += 1;
      const rhs = this.parseFactor();
      value = t.op === "*" ? value * rhs : value / rhs;
    }
    return value;
  }

  private parseFactor(): number {
    const t = this.peek();
    if (!t) throw new Error("unexpected end");
    if (t.kind === "op" && t.op === "-") {
      this.pos += 1;
      return -this.parseFactor();
    }
    if (t.kind === "op" && t.op === "(") {
      this.pos += 1;
      const v = this.parseExpression();
      const close = this.peek();
      if (!close || close.kind !== "op" || close.op !== ")") throw new Error("expected )");
      this.pos += 1;
      return v;
    }
    if (t.kind === "num") {
      this.pos += 1;
      return t.value;
    }
    if (t.kind === "ref") {
      this.pos += 1;
      const raw = this.data[t.key];
      return typeof raw === "number" && Number.isFinite(raw) ? raw : 0;
    }
    throw new Error("bad factor");
  }
}
