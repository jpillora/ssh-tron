## ssh-tron

Multiplayer [Tron](http://www.classicgamesarcade.com/game/21670/tron-game.html) over SSH, written in Go

![tron](https://rawgit.com/jpillora/ssh-tron/master/demo.gif)

### Server Install

#### From Source

```
$ go get -v github.com/jpillora/ssh-tron
$ ssh-tron
```

#### Unix Binaries

```
$ curl -"#" -L https://github.com/jpillora/ssh-tron/releases/download/2.1.2/tron_darwin_amd64.gz | gzip -d > tron
#                                     ...change OS and ARCH as necessary...      linux  368
$ chmod +x tron
$ ./tron
```

Optionally move to PATH

```
$ mv tron /usr/local/bin/
$ tron
```

https://github.com/jpillora/ssh-tron/releases

#### Windows

*It may work under Windows though it's currently untested*

### Usage

Server:

```
$ tron --help
Usage of tron:
  -deaths=10: Maximum number of deaths before being kicked
  -delay=2000: Respawn delay (in ms)
  -height=60: Height of the game world
  -players=4: Maximum number of simultaneous players
  -port=2200: Port to listen for TCP connections on
  -speed=25: Game tick interval (in ms)
  -width=60: Width of the game world

$ tron
tron: game started (#4 player slots)
server: server up - join at
server:   ○ ssh 127.0.0.1 -p 2200
server:   ○ ssh 10.7.0.108 -p 2200
server:   ○ ssh 192.168.136.1 -p 2200
server:   ○ ssh 172.16.4.1 -p 2200
```

Players:

```
$ ssh 10.7.0.108 -p 2200
```

*Press `Enter` to spawn*

### Known Client Issues

* Appears best with a dark terminal background
* The refresh rate is quite high, so you'll need a low latency connection to the server to play effectively (approximately less than 25ms).
* Only works on operating systems with [braille unicode characters (e.g. "⠶" and "⠛")](http://en.wikipedia.org/wiki/Braille_Patterns#Chart) installed. Operating systems lacking this character set will cause the walls to render as the missing glyph (square or diamond).

### Todo

* Support multi-core (Fix race conditions)
* Optimise game calculations
* Optimise network
* `SPACE` to invert colours
* Add "kills" option (end the game once someone reaches `kills`)
* Add "all players reset on any death" option.
* Extract code to produce a generic 2D multi-player game engine
	* Bomber man
	* Dungeon explorer

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