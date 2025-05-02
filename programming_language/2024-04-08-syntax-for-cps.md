---
date: 2024-04-08
original: https://takoeight0821.hatenablog.jp/entry/2024/04/08/193449
---

# 継続渡し・コールバックを読みやすくする言語機能たち（Koka・Gleam・Roc）

継続渡しスタイル、あるいはコールバック関数は非常に強力なテクニックだ。
例えばJavaScriptでは、非同期処理を扱う`.then`メソッドが有名どころだろう。

```javascript
fetch("http://example.com/movies.json")
  .then((response) => response.json())
  .then((movies) => console.log(movies))
```

継続渡しスタイルは読みにくい。そこで、JavaScriptではasync構文が導入されている。

```javascript
const response = await fetch("http://example.com/movies.json");
const movies = await response.json();
console.log(movies);
```

awaitの振る舞いは、以下のような読み替えルールがあると考えると理解しやすい。

```javascript
const X = await P; E;
=>
P.then((X) => E);
```

awaitは、継続渡しスタイルの非同期プログラムを、あたかも直接スタイルかのように書くための言語機能だ、と解釈できる。
プログラミング言語の中には、より汎用的に継続渡しスタイルを直接スタイルに変える言語機能を持つものがある。

[Koka](https://koka-lang.github.io/koka/doc/index.html)には、`with`構文がある。
例えば、1から10までの整数を標準出力に書き出すKokaプログラムは以下のようになる：

```koka
list(1,10).foreach(fn (x) {
  println(x)
})
```

`with`構文を使うと、以下のように書ける。

```koka
with x <- list(1,10).foreach
println(x)
```

`with`の読み替えルールは以下のようになる：

```
with X <- F(A, ...)
E
=>
F(A, ..., fn (X) { E })
```

Kokaの名前は日本語の「効果」に由来する。その名が示す通り、Kokaは代数的効果（Algebraic effects）をサポートしている。
代数的効果はざっくり言えば「すごく高機能な例外」だ。例えば、以下のKokaプログラムは、0除算エラー（raiseエフェクト）を起こしうる関数`divide`を定義している。

```koka
fun divide( x : int, y : int ) : raise int
  if y==0 then raise("div-by-zero") else x / y
```

`raise`の振る舞いを自由に後づけできるのが代数的効果の特徴だ。次のプログラムは、例外をもみ消して定数`42`に評価されたものとする。`handler`構文で各エフェクトの実装を与えている。次のプログラムは最終的に`50`を返す。

```koka
(hander {
  ctl raise(msg) { 42 }
})(fn () {
  8 + divide(1, 0)
})
```

`hander {...}`は、エフェクトを起こしうる処理をコールバック関数として受け取る関数になっている。
コールバック関数を受け取るということはつまり、`with`を使うともっとスマートに書ける。

```koka
with handler { ctl raise(msg) { 42 } }
8 + divide(1, 0)
```

[Gleam](https://gleam.run/)にも同様の振る舞いをする`use`構文がある。
[Use \- The Gleam Language Tour](https://tour.gleam.run/advanced-features/use/)から、`use`を使ったコード例を引用する：

```gleam
pub fn without_use() {
  result.try(get_username(), fn(username) {
    result.try(get_password(), fn(password) {
      result.map(log_in(username, password), fn(greeting) {
        greeting <> ", " <> username
      })
    })
  })
}

pub fn with_use() {
  use username <- result.try(get_username())
  use password <- result.try(get_password())
  use greeting <- result.map(log_in(username, password))
  greeting <> ", " <> username
}
```

`result.try`は、成功か失敗を表す`result`値と、成功したなら実行されるコールバック関数を受け取り、最初の引数が成功値ならコールバックを適用、失敗値ならそれをそのまま返す。
`use`構文はKokaの`with`と同様の振る舞いをするので、`with_use()`のような書き方ができる。

[Roc](https://www.roc-lang.org/)はHaskellに似た軽量な構文を持つプログラミング言語だ。
Rocで標準入出力を扱うプログラムを書くと、以下のようになる。

```roc
main =
    await (Stdout.line "Type something press Enter:") \_ ->
        await Stdin.line \input ->
            Stdout.line "Your input was: $(Inspect.toStr input)"
```

`await`はJavaScriptのそれとは異なり、単なる関数である。第一引数に実行したいタスクを、第二引数にタスクの結果を処理するコールバック関数を取る。`\input -> ...`は無名関数だ。

Rocでは、`<-`を使って継続渡しスタイルを直接スタイルに書き換える。

```roc
main =
    _ <- await (Stdout.line "Type something press Enter:")
    input <- await Stdin.line

    Stdout.line "Your input was: $(Inspect.toStr input)"
```

JavaScriptに見られる`async`構文は、様々な言語で導入されている。
いくつかの言語では、より柔軟な形で`async`のような構文を定義できるようになっている。
例えばF#ではcomputation expressionが、Haskellではdo構文とモナドが使われている。
これらの言語機能は強力な反面、言語への導入に少々ハードルがある。

`async`構文が本当にやりたいことは継続渡しスタイルを直接スタイルのように書くことだ、と思うと、もっと単純な解決策がある。それがKokaの`with`やGleamの`use`構文だ。
あるいは、`with`や`use`と同じように、`async`構文は継続渡しスタイルに立ち向かうための道具だ、という見方もできる。