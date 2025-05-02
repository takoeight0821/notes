# Compiler

<details>
<summary># 関数適用は後置演算子 | 2025-04-12</summary>

<a href="2025-04-12-parsing_application.md">link</a>

## 関数適用は後置演算子

再帰下降構文解析で、以下のような関数適用をうまくパースしたい。

```scala
List(1, 2, 3).map(x => add(1)(x))
```

こんな文法を考えると、うまくいきそうに思う。

```bnf
term = variable | literal ;
expr = term
     | expr "(" expr ("," expr)* ")"
     | expr "." ident
     ;
```

再帰下降構文解析を実装するときは、まず左再帰除去を行う。
さっきの文法を左再帰除去するとこんな感じ。

```ebnf
expr = term exprTail ;

exprTail = ε
         | "(" expr ("," expr)* ")" exprTail
         | "." ident exprTail
         ;
```

この文法を実装すると、だいたいこんな感じ。
簡単だけど、↑の文法と↓の文法が一対一で対応している！って感じはしない。

```go
func (p *Parser) expr() Node {
    term := p.term()
    return p.exprTail(term)
}

func (p *Parser) exprTail(expr Node) Node {
    if p.peek() == "(" {
        arguments := p.arguments()
        return p.exprTail(Apply { expr, arguments })
    } else if p.peek() == "." {
        name := p.ident()
        return p.exprTail(Project { expr, name })
    }
    return expr
}
```

改めてお題に戻る。
↓のコードをグッと睨むと、関数適用`(1, 2, 3)`やフィールドアクセス`.map`が、**後置演算子**に見えてこないだろうか。
後置演算子は、`i++`みたいなやつ。見えてこない？

```scala
List(1, 2, 3).map(x => add(1)(x))
```

見えたとします。関数適用は後置演算子だ！って考えで文法を書くとこうなる。

```ebnf
expr = term exprTail+ ;

exprTail = "(" expr ("," expr)* ")"
         | "." ident
         ;
```

さっきの文法と比べると、意味は同じだけどスッキリしている。

```diff
-expr = term exprTail ;
+expr = term exprTail+ ;

-exprTail = ε
-      | "(" expr ("," expr)* ")" exprTail
-      | "." ident exprTail
-      ;
+exprTail = "(" expr ("," expr)* ")"
+         | "." ident
+         ;
```

実装もしやすい。

```go
func (p *Parser) expr() Node {
    term := p.term()
    for p.peek() == "(" || p.peek() == "." {
        term = p.exprTail(term)
    }

    return term
}

func (p *Parser) exprTail(expr Node) Node {
    if p.peek() == "(" {
        arguments := p.arguments()
        return Apply { expr, arguments }
    } else if p.peek() == "." {
        name := p.ident()
        return Project { expr, name }
    }
}
```

以前のコードでは`exprTail`は再帰関数だった。
一方、今回のコードでは`expr`内でループ呼び出ししている。
しかも「関数適用は後置演算子」と思いながら直感的に書いた文法と一対一で対応している！

最近は「関数適用は後置演算子」戦略でパーサーを書くことが多い。
多分、これを発展させるとPrattパーサーみたいな話が出てくるのかな。
</details>

