# logmon-go
logmon by go lang

## 概要
設定ファイルで指定されたFileを監視し、正規表現で指定した文字パターンが現われると、指定したコマンドが実行される。

## Usage
```
$ logmon-go -c sample.conf
```

## Config

```
#監視ファイル
:/var/log/nginx/error.log
#監視対象Regexp ()で囲む必要
(ERROR|Error)
#監視対象除外Regexp []で囲む必要。監視対象Regexpにひっかかっても除外したいパターンを指定。指定しなくてもよい
[SSL_BYTES_TO_CIPHER_LIST]
#監視対象Regexpにマッチかつ監視対象外Regexpにマッチしなかった場合に実行されるコマンド. <%%%%>は一致した文字列に置き換えられる。
#<%%%%>を含む箇所はエスケープさせるため'(シングルクォート)で囲む
echo  -e 'ERROR\n<%%%%>\n' | mail -s "nginx error" takeshy
#複数の監視ファイルがある場合は上記を繰り替えす
:/var/www/app/shared/log/unicorn.log
(ERROR|FATAL)
echo  -e 'ERROR\n<%%%%>\n' | mail -s "unicorn error" takeshy
```

## Installation

```
go get github.com/takeshy/logmon-go
```


## License

MIT

## Author

Takeshi Morita
