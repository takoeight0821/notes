---
date: 2025-05-02
original: https://github.com/takoeight0821/notes/blob/main/compiler/2025-05-02-assembly-as-embeded-dsl.md
---
# Embeded DSL としてアセンブラを実装する

人生は長く複雑なので、アセンブラのないCPUでプログラムを書くこともあります。
そういうときはだいたい頑張ってハンドアセンブルするわけですが、ちょっとプログラミングをがんばればなんとかなったりもします。
今回は、組み込みDSL（Embeded DSL）と呼ばれるテクニックを使って、手早くアセンブラ（しかも超高級なやつ）をでっち上げてみます。

## 組み込みDSL（Embeded DSL）とは

組み込みDSLとは、関数やクラスをうまく使って、あたかも専用の言語かのように書けるようにライブラリ・フレームワークを作るテクニックです。
例えば、以下のようなDSLがあります。

- Vagrant : VMの構成をRubyで書く

```ruby
Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/bionic64"
  config.vm.network "private_network", type: "dhcp"
  config.vm.provider "virtualbox" do |vb|
    vb.memory = "1024"
    vb.cpus = 2
  end
end
```

- Gradle : ビルド手順をGroovy/Kotlinで書く

```kotlin
plugins {
    // Apply the application plugin to add support for building a CLI application in Java.
    application
}
dependencies {
    // Use JUnit Jupiter for testing.
    testImplementation(libs.junit.jupiter)

    testRuntimeOnly("org.junit.platform:junit-platform-launcher")

    // This dependency is used by the application.
    implementation(libs.guava)
}
application {
    // Define the main class for the application.
    mainClass = "org.example.App"
}
```

- Jest : テスト内容をJavaScriptで書く

```javascript
const sum = require('./sum');

test('adds 1 + 2 to equal 3', () => {
  expect(sum(1, 2)).toBe(3);
});
```

TODO: EDSLについてまとめたあと、コンパイラのT字図を示しながら、EDSLである種のコンパイラ・インタプリタを自動実装できることを示す。