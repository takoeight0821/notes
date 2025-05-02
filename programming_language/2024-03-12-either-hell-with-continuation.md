---
date: 2024-03-12
original: https://takoeight0821.hatenablog.jp/entry/2024/03/12/150448
---
# 継続モナドで立ち向かうローンパターンとEither地獄

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

## 参考文献
* [ローンパターン \- haskell\-shoen](https://scrapbox.io/haskell-shoen/%E3%83%AD%E3%83%BC%E3%83%B3%E3%83%91%E3%82%BF%E3%83%BC%E3%83%B3)
* [Why would you use ContT?](https://ro-che.info/articles/2019-06-07-why-use-contt)
  + `ContT`を使って継続渡しスタイルをdo記法に書き換える例を紹介している。
* [Lysxia \- The reasonable effectiveness of the continuation monad](https://blog.poisson.chat/posts/2019-10-26-reasonable-continuations.html)
  + 継続モナドを使って他の様々なモナドを実装する記事。継続モナドの強力さを示す好例。
* [ContT を使ってコードを綺麗にしよう！](https://github.com/e-bigmoon/haskell-blog/blob/a737b9549130ea61f7a299628f3350c00326ac03/posts/2018/06-26-cont-param.md)（元サイトリンク切れのため、Githubで公開されてるMarkdown原稿をリンク）
  + 本記事よりもイカした手法を紹介している。
* [fallibleというパッケージをリリースしました \- Haskell\-jp](https://haskell.jp/blog/posts/2019/fallible.html)
  + ↑の記事をEitherに拡張したもの。

## 追記というか余談

[haskell \- Monadic function of \`\(a \-> m \(Either e b\)\) \-> Either e a \-> m \(Either e b\)\`? \- Stack Overflow](https://stackoverflow.com/questions/73354040/monadic-function-of-a-m-either-e-b-either-e-a-m-either-e-b)によると、`with`はもっと抽象化できるらしい。

```haskell
with :: (Monad m, Monad f, Traversable f) => m (f a) -> ContT (f b) m a
with m = ContT \k -> do
  x <- m
  join <$> traverse k x
```

この`with`は`Maybe`にも対応している。対応しているが、ちょっとやりすぎな気もする。