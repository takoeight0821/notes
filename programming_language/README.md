# Programming Language

<details>
<summary># 継続モナドで立ち向かうローンパターンとEither地獄 | 2024-03-12</summary>

<a href="2024-03-12-either-hell-with-continuation.md">link</a>

## 継続モナドで立ち向かうローンパターンとEither地獄

Haskellでファイルなどのリソースの解放を保証するテクニックとして、ローンパターン（Loan Pattern）がある。`withFile :: FilePath -> IOMode -> (Handle -> IO r) -> IO r`などがその例だ。
ローンパターンによる関数を複数使ったプログラムは、無名関数のネストが深くなる。

```haskell
main = do
  withFile "src.txt" ReadMode \src ->
    withFile "dst.txt" WriteMode \dst ->
      ...
```

この問題には、継続モナド`ContT`を使ったきれいな解決策が知られている。

```haskell
main = evalContT do
  src <- ContT $ withFile "src.txt" ReadMode
  dst <- ContT $ withFile "dst.txt" WriteMode
  ...
```

ミソは、ContTを使うことで、継続渡しスタイルをdo記法に変換できるところにある。

このアイディアを更に深堀りしてみよう。
設定ファイルを読み込みパースする関数`parseConfig`と、`Config`のあるフィールドを取得する関数`getField`があるとする。
設定ファイルを読み込んでフィールド`language`を取得し、アプリケーションの言語を変更する処理は次のように書ける。

```haskell
parseConfig :: MonadIO m => FilePath -> m (Either String Config)
getField :: MonadIO m => Config -> m (Either String Value)

updateLanguage :: (MonadIO m, MonadState Env) => m ()
updateLanguage = do
  ecfg <- parseConfig "app.cfg"
  case ecfg of
    Left err -> error err
    Right cfg -> do
      elang <- getField cfg "language"
      case elang of
        Left err -> error err
        Right lang -> modify \env -> env { language = lang }
```

`Left`と`Right`のパターンマッチが繰り返されている。
こういう地獄に落ちると人は「おお神よ！try-catch構文はどこへ行ってしまったのです！」という気分になり、`ExceptT`などの例外モナドを使って`parseConfig`と`getField`を書き直したくなる[^1]。
[^1]: 例えば`parseConfig :: (MonadIO m, MonadError String m) => FilePath -> m Config`。モナドを使うときはmtlスタイルで書くようにすれば地獄行きをある程度防止できる。`Either`が例外モナドであることを失念しないように気をつけよう。
書き直せるなら万々歳だが、悪しき`parseConfig`がすでにコードの深いところまで根を張っていて、そう簡単には修正できないことも多い。

`ContT`は、こんなときに助けになる。まずは、継続渡しスタイルを使って`updateLanguage`を書き換えよう。

```haskell
updateLanguage :: (MonadIO m, MonadState Env) => m ()
updateLanguage = do
  with (parseConfig "app.cfg") \cfg ->
    with (getField cfg "language") \lang ->
      modify \env -> env { language = lang }

with :: Monad m => m (Either String a) -> (a -> m (Either String b)) -> m (Either String b)
with m k = do
  ea <- m
  case ea of
    Left err -> pure $ Left err
    Right a -> k a
```
（サラッと書いてしまったが、こういう書き換えは難しい。コツを掴めるまでしばらくかかるが、できるようになると色々便利。）

これで`Left`と`Right`のパターンマッチを一つにできた。あとは`ContT`を使ってdo記法に戻せばいい。
`with`を`ContT`でラップしよう。

```haskell
updateLanguage :: (MonadIO m, MonadState Env) => m ()
updateLanguage = evalContT do
  cfg <- with (parseConfig "app.cfg")
  lang <- with (getField cfg "language")
  modify \env -> env { language = lang }

with :: Monad m => m (Either String a) -> ContT (Either String b) m a
with m = ContT \k -> do
  ea <- m
  case ea of
    Left err -> pure $ Left err
    Right a -> k a
```

ややこしいコードを上手く継続渡しスタイルに落としこめれば、`ContT`を使ったシンプルなdo記法にリファクタリングできる。
`ContT`自体が少々ややこしいので乱用は禁物だが、うまく使えば最小限の変更でプログラムがグッと読みやすくなる。

### 参考文献
* [ローンパターン \- haskell\-shoen](https://scrapbox.io/haskell-shoen/%E3%83%AD%E3%83%BC%E3%83%B3%E3%83%91%E3%82%BF%E3%83%BC%E3%83%B3)
* [Why would you use ContT?](https://ro-che.info/articles/2019-06-07-why-use-contt)
  + `ContT`を使って継続渡しスタイルをdo記法に書き換える例を紹介している。
* [Lysxia \- The reasonable effectiveness of the continuation monad](https://blog.poisson.chat/posts/2019-10-26-reasonable-continuations.html)
  + 継続モナドを使って他の様々なモナドを実装する記事。継続モナドの強力さを示す好例。
* [ContT を使ってコードを綺麗にしよう！](https://github.com/e-bigmoon/haskell-blog/blob/a737b9549130ea61f7a299628f3350c00326ac03/posts/2018/06-26-cont-param.md)（元サイトリンク切れのため、Githubで公開されてるMarkdown原稿をリンク）
  + 本記事よりもイカした手法を紹介している。
* [fallibleというパッケージをリリースしました \- Haskell\-jp](https://haskell.jp/blog/posts/2019/fallible.html)
  + ↑の記事をEitherに拡張したもの。

### 追記というか余談

[haskell \- Monadic function of \`\(a \-> m \(Either e b\)\) \-> Either e a \-> m \(Either e b\)\`? \- Stack Overflow](https://stackoverflow.com/questions/73354040/monadic-function-of-a-m-either-e-b-either-e-a-m-either-e-b)によると、`with`はもっと抽象化できるらしい。

```haskell
with :: (Monad m, Monad f, Traversable f) => m (f a) -> ContT (f b) m a
with m = ContT \k -> do
  x <- m
  join <$> traverse k x
```

この`with`は`Maybe`にも対応している。対応しているが、ちょっとやりすぎな気もする。
</details>

<details>
<summary># 継続渡し・コールバックを読みやすくする言語機能たち（Koka・Gleam・Roc） | 2024-04-08</summary>

<a href="2024-04-08-syntax-for-cps.md">link</a>

## 継続渡し・コールバックを読みやすくする言語機能たち（Koka・Gleam・Roc）

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
</details>

<details>
<summary># Programming Language | </summary>

<a href="index.md">link</a>

## Programming Language

<details>
<summary># 継続モナドで立ち向かうローンパターンとEither地獄 | 2024-03-12</summary>

<a href="2024-03-12-either-hell-with-continuation.md">link</a>

### 継続モナドで立ち向かうローンパターンとEither地獄

Haskellでファイルなどのリソースの解放を保証するテクニックとして、ローンパターン（Loan Pattern）がある。`withFile :: FilePath -> IOMode -> (Handle -> IO r) -> IO r`などがその例だ。
ローンパターンによる関数を複数使ったプログラムは、無名関数のネストが深くなる。

```haskell
main = do
  withFile "src.txt" ReadMode \src ->
    withFile "dst.txt" WriteMode \dst ->
      ...
```

この問題には、継続モナド`ContT`を使ったきれいな解決策が知られている。

```haskell
main = evalContT do
  src <- ContT $ withFile "src.txt" ReadMode
  dst <- ContT $ withFile "dst.txt" WriteMode
  ...
```

ミソは、ContTを使うことで、継続渡しスタイルをdo記法に変換できるところにある。

このアイディアを更に深堀りしてみよう。
設定ファイルを読み込みパースする関数`parseConfig`と、`Config`のあるフィールドを取得する関数`getField`があるとする。
設定ファイルを読み込んでフィールド`language`を取得し、アプリケーションの言語を変更する処理は次のように書ける。

```haskell
parseConfig :: MonadIO m => FilePath -> m (Either String Config)
getField :: MonadIO m => Config -> m (Either String Value)

updateLanguage :: (MonadIO m, MonadState Env) => m ()
updateLanguage = do
  ecfg <- parseConfig "app.cfg"
  case ecfg of
    Left err -> error err
    Right cfg -> do
      elang <- getField cfg "language"
      case elang of
        Left err -> error err
        Right lang -> modify \env -> env { language = lang }
```

`Left`と`Right`のパターンマッチが繰り返されている。
こういう地獄に落ちると人は「おお神よ！try-catch構文はどこへ行ってしまったのです！」という気分になり、`ExceptT`などの例外モナドを使って`parseConfig`と`getField`を書き直したくなる[^1]。
[^1]: 例えば`parseConfig :: (MonadIO m, MonadError String m) => FilePath -> m Config`。モナドを使うときはmtlスタイルで書くようにすれば地獄行きをある程度防止できる。`Either`が例外モナドであることを失念しないように気をつけよう。
書き直せるなら万々歳だが、悪しき`parseConfig`がすでにコードの深いところまで根を張っていて、そう簡単には修正できないことも多い。

`ContT`は、こんなときに助けになる。まずは、継続渡しスタイルを使って`updateLanguage`を書き換えよう。

```haskell
updateLanguage :: (MonadIO m, MonadState Env) => m ()
updateLanguage = do
  with (parseConfig "app.cfg") \cfg ->
    with (getField cfg "language") \lang ->
      modify \env -> env { language = lang }

with :: Monad m => m (Either String a) -> (a -> m (Either String b)) -> m (Either String b)
with m k = do
  ea <- m
  case ea of
    Left err -> pure $ Left err
    Right a -> k a
```
（サラッと書いてしまったが、こういう書き換えは難しい。コツを掴めるまでしばらくかかるが、できるようになると色々便利。）

これで`Left`と`Right`のパターンマッチを一つにできた。あとは`ContT`を使ってdo記法に戻せばいい。
`with`を`ContT`でラップしよう。

```haskell
updateLanguage :: (MonadIO m, MonadState Env) => m ()
updateLanguage = evalContT do
  cfg <- with (parseConfig "app.cfg")
  lang <- with (getField cfg "language")
  modify \env -> env { language = lang }

with :: Monad m => m (Either String a) -> ContT (Either String b) m a
with m = ContT \k -> do
  ea <- m
  case ea of
    Left err -> pure $ Left err
    Right a -> k a
```

ややこしいコードを上手く継続渡しスタイルに落としこめれば、`ContT`を使ったシンプルなdo記法にリファクタリングできる。
`ContT`自体が少々ややこしいので乱用は禁物だが、うまく使えば最小限の変更でプログラムがグッと読みやすくなる。

#### 参考文献
* [ローンパターン \- haskell\-shoen](https://scrapbox.io/haskell-shoen/%E3%83%AD%E3%83%BC%E3%83%B3%E3%83%91%E3%82%BF%E3%83%BC%E3%83%B3)
* [Why would you use ContT?](https://ro-che.info/articles/2019-06-07-why-use-contt)
  + `ContT`を使って継続渡しスタイルをdo記法に書き換える例を紹介している。
* [Lysxia \- The reasonable effectiveness of the continuation monad](https://blog.poisson.chat/posts/2019-10-26-reasonable-continuations.html)
  + 継続モナドを使って他の様々なモナドを実装する記事。継続モナドの強力さを示す好例。
* [ContT を使ってコードを綺麗にしよう！](https://github.com/e-bigmoon/haskell-blog/blob/a737b9549130ea61f7a299628f3350c00326ac03/posts/2018/06-26-cont-param.md)（元サイトリンク切れのため、Githubで公開されてるMarkdown原稿をリンク）
  + 本記事よりもイカした手法を紹介している。
* [fallibleというパッケージをリリースしました \- Haskell\-jp](https://haskell.jp/blog/posts/2019/fallible.html)
  + ↑の記事をEitherに拡張したもの。

#### 追記というか余談

[haskell \- Monadic function of \`\(a \-> m \(Either e b\)\) \-> Either e a \-> m \(Either e b\)\`? \- Stack Overflow](https://stackoverflow.com/questions/73354040/monadic-function-of-a-m-either-e-b-either-e-a-m-either-e-b)によると、`with`はもっと抽象化できるらしい。

```haskell
with :: (Monad m, Monad f, Traversable f) => m (f a) -> ContT (f b) m a
with m = ContT \k -> do
  x <- m
  join <$> traverse k x
```

この`with`は`Maybe`にも対応している。対応しているが、ちょっとやりすぎな気もする。
</details>

<details>
<summary># 継続渡し・コールバックを読みやすくする言語機能たち（Koka・Gleam・Roc） | 2024-04-08</summary>

<a href="2024-04-08-syntax-for-cps.md">link</a>

### 継続渡し・コールバックを読みやすくする言語機能たち（Koka・Gleam・Roc）

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
</details>

</details>

