## tron

Multiplayer Tron over SSH, written in Go

### Install

#### Unix Binaries:

```
# not working yet

$ curl -L https://github.com/jpillora/tron/releases/download/1.0.0/tron_darwin_amd64.gz | gzip -d > tron
                                                                OR      linux  368
$ chmod +x tron
$ ./tron
$ # ....
$ #optional
$ mv tron /usr/local/bin/
$ tron
```

https://github.com/jpillora/tron/releases

#### Windows:

*It may work under Windows though it's currently untested*

#### Source:

```
go get -v github.com/jpillora/tron
```

### Usage

Server:

```
$ tron
tron: game started (#6 player slots)
server: server up - join at
server:   ○ ssh 127.0.0.1 -p 2200
server:   ○ ssh 10.7.0.108 -p 2200
server:   ○ ssh 25.24.208.189 -p 2200
server:   ○ ssh 192.168.136.1 -p 2200
server:   ○ ssh 172.16.4.1 -p 2200
```

Players:

```
$ ssh 10.7.0.108 -p 2200
```

*Press `Enter` to spawn*

#### MIT License

Copyright © 2014 &lt;dev@jpillora.com&gt;

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.