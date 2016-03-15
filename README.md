## ssh-tron

Multiplayer [Tron](http://www.thepcmanwebsite.com/media/flash_tron/) (Light Cycles) over SSH, written in Go - ported from a telnet version I'd previously written in Node many years ago.

![tron](https://rawgit.com/jpillora/ssh-tron/master/demo.gif)

### Demo

```
ssh tron.jpillora.com
```

:warning: This server is in Australia

### Install

#### Docker

```sh
#run on the host's port 22
docker run --name=tron -d -p 22:2200 jpillora/ssh-tron
#run on the host's port 22 and save player scores
touch /path/to/tron.db
docker run --name=tron -d -v /path/to/tron.db:/tron.db -p 22:2200 jpillora/ssh-tron --db-location /tron.db
```

#### Binaries

See [the latest release](https://github.com/jpillora/ssh-tron/releases/latest) or install it now with `curl i.jpillora.com/ssh-tron! | bash`

#### From Source

```
$ go get -v github.com/jpillora/ssh-tron
```

### Usage

Server:

```
$ tron --help

  Usage: ssh-tron [options]

  Options:
  --port, -p           Port to listen for TCP connections on (default 2200)
  --width, -w          Width of the game world (default 60)
  --height, -h         Height of the game world (default 60)
  --max-players, -m    Maximum number of simultaneous players (default 6)
  --game-speed, -g     Game tick interval, basically controls how fast each
                       player moves (default 40ms)
  --respawn-delay, -r  The time a player must wait before being able to
                       respawn (default 2s)
  --db-location, -d    Location of tron.db, stores game score and config (default /tmp/tron.db)
  --db-reset           Reset all scores in the database
  --help
  --version, -v

  Author:
    jpillora

  Version:
    3.0.1

  Read more:
    https://github.com/jpillora/ssh-tron

$ tron
tron: game started (#6 player slots)
server: up - join at
server:   ○ ssh 127.0.0.1 -p 2200
server:   ○ ssh 172.27.1.78 -p 2200
server:   ○ ssh 192.168.136.1 -p 2200
server:   ○ ssh 172.16.4.1 -p 2200
server: fingerprint - 5e:6b:8f:f5:39:af:57:84:3c:5a:a5:32:d7:41:04:b8
```

Players:

```
$ ssh 172.27.1.78 -p 2200
```

*Press `Enter` to spawn*

### Known Client Issues

* Appears best with a dark terminal background
* The refresh rate is quite high, so you'll need a low latency connection to the server to play effectively (in essense, you want your latency to be lower the game speed - which has a default of 40ms/tick).
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
